package kjobrunner

import (
	"errors"
	"fmt"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// Errors
var (
	ErrNotExists = errors.New("kjobrunner: not exists")
)

// Runner wraps k8s job with simple api
type Runner struct {
	name   string
	client *kubernetes.Clientset
	labels map[string]string
	ns     string
}

const defaultName = "kjobrunner"

// New creates new runner
func New(name string, client *kubernetes.Clientset, namespace string) *Runner {
	if name == "" {
		name = defaultName
	}

	var r Runner
	r.name = name
	r.client = client
	r.labels = map[string]string{
		"scheduler": name,
	}
	r.ns = namespace
	return &r
}

type RunOption struct {
	Name     string
	Image    string
	Envs     *Envs
	Args     []string
	Replicas int32
}

func (r *Runner) Run(opt *RunOption) error {
	labels := cloneLabels(r.labels)
	labels["name"] = opt.Name

	if opt.Replicas <= 0 {
		opt.Replicas = 1
	}

	_, err := r.client.BatchV1().Jobs(r.ns).Create(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:   opt.Name,
			Labels: labels,
		},
		Spec: batchv1.JobSpec{
			Parallelism: &opt.Replicas,
			Completions: &opt.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "container",
							Image: opt.Image,
							Env:   opt.Envs.envVars(),
							Args:  opt.Args,
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	})
	return err
}

func (r *Runner) List() ([]string, error) {
	list, err := r.client.BatchV1().Jobs(r.ns).List(metav1.ListOptions{
		LabelSelector: "scheduler=" + r.name,
	})
	if err != nil {
		return nil, err
	}

	var rs []string
	for _, job := range list.Items {
		rs = append(rs, job.Name)
	}
	return rs, nil
}

func (r *Runner) Delete(name string) error {
	exists, err := r.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotExists
	}

	policy := metav1.DeletePropagationBackground
	return r.client.BatchV1().Jobs(r.ns).Delete(name, &metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
}

func (r *Runner) Exists(name string) (bool, error) {
	job, err := r.client.BatchV1().Jobs(r.ns).Get(name, metav1.GetOptions{})
	if err, ok := err.(*apierrors.StatusError); ok {
		if err.ErrStatus.Reason == metav1.StatusReasonNotFound {
			return false, nil
		}
	}
	if err != nil {
		return false, err
	}

	l := job.GetLabels()
	if l == nil {
		return false, nil
	}
	if l["scheduler"] != r.name {
		return false, nil
	}

	return true, nil
}

func (r *Runner) Wait(name string) error {
	return wait.PollImmediate(2*time.Second, 1*time.Hour, func() (bool, error) {
		job, err := r.client.BatchV1().Jobs(r.ns).Get(name, metav1.GetOptions{})
		if err, ok := err.(*apierrors.StatusError); ok {
			if err.ErrStatus.Reason == metav1.StatusReasonNotFound {
				return true, ErrNotExists
			}
		}
		if err != nil {
			return true, err
		}

		if job.Status.CompletionTime.IsZero() {
			return false, nil
		}
		return true, nil
	})
}

// Cleanup deletes all success jobs
func (r *Runner) Cleanup() error {
	client := r.client.BatchV1().Jobs(r.ns)

	list, err := client.List(metav1.ListOptions{
		LabelSelector: "scheduler=" + r.name,
	})
	if err != nil {
		return err
	}

	policy := metav1.DeletePropagationBackground
	for _, job := range list.Items {
		if job.Status.CompletionTime.IsZero() {
			continue
		}

		err = client.Delete(job.Name, &metav1.DeleteOptions{
			PropagationPolicy: &policy,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) Logs(name string) (string, error) {
	client := r.client.CoreV1().Pods(r.ns)

	labels := cloneLabels(r.labels)
	labels["name"] = name

	pods, err := client.List(metav1.ListOptions{
		LabelSelector: labelsToSelector(labels),
	})
	if err != nil {
		return "", err
	}
	if len(pods.Items) == 0 {
		return "", ErrNotExists
	}

	logs, err := client.GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{}).DoRaw()
	if err != nil {
		return "", err
	}
	return string(logs), nil
}

func cloneLabels(src map[string]string) map[string]string {
	dst := make(map[string]string)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func labelsToSelector(labels map[string]string) string {
	var xs []string
	for name, value := range labels {
		xs = append(xs, fmt.Sprintf("%s=%s", name, value))
	}
	return strings.Join(xs, ", ")
}
