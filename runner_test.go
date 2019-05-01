package kjobrunner_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	. "github.com/acoshift/kjobrunner"
)

func newLocalKubernetes() *kubernetes.Clientset {
	token := os.Getenv("TOKEN")

	return kubernetes.NewForConfigOrDie(&rest.Config{
		Host:        "localhost:6443",
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	})
}

func TestRunner(t *testing.T) {
	t.Parallel()

	runner := New("runner", newLocalKubernetes(), "kjobrunner")

	{
		exists, err := runner.Exists("notexists")
		assert.NoError(t, err)
		assert.False(t, exists)
	}

	{
		err := runner.Run(&RunOption{
			Name:     "hello",
			Image:    "hello-world",
			Replicas: 2,
		})
		assert.NoError(t, err)
	}

	{
		exists, err := runner.Exists("hello")
		assert.NoError(t, err)
		assert.True(t, exists)
	}

	{
		list, err := runner.List()
		assert.NoError(t, err)
		if assert.Len(t, list, 1) {
			assert.Equal(t, "hello", list[0])
		}
	}

	{
		err := runner.Wait("hello")
		assert.NoError(t, err)
	}

	{
		logs, err := runner.Logs("hello")
		assert.NoError(t, err)
		assert.True(t, strings.Contains(logs, "Hello from Docker!"))
	}

	{
		err := runner.Delete("hello")
		assert.NoError(t, err)
	}
}

func TestCleanup(t *testing.T) {
	t.Parallel()

	// run hello-1
	// run hello-2
	// wait for hello-1 and hello-2 to completed
	// run hello-3
	// call cleanup
	// hello-1 and hello-2 must be deleted
	// delete hello-3 to cleanup resource

	runner := New("cleanup", newLocalKubernetes(), "kjobrunner")

	err := runner.Run(&RunOption{
		Name:  "hello-1",
		Image: "hello-world",
	})
	assert.NoError(t, err)

	err = runner.Run(&RunOption{
		Name:  "hello-2",
		Image: "hello-world",
	})
	assert.NoError(t, err)

	err = runner.Wait("hello-1")
	assert.NoError(t, err)
	err = runner.Wait("hello-2")
	assert.NoError(t, err)

	err = runner.Run(&RunOption{
		Name:  "hello-3",
		Image: "hello-world",
	})
	assert.NoError(t, err)

	err = runner.Cleanup()
	assert.NoError(t, err)

	list, err := runner.List()
	assert.NoError(t, err)
	if assert.Len(t, list, 1) {
		assert.Equal(t, "hello-3", list[0])
	}

	err = runner.Delete("hello-3")
	assert.NoError(t, err)
}
