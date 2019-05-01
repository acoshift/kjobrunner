package kjobrunner

import (
	corev1 "k8s.io/api/core/v1"
)

type env struct {
	name  string
	value string
}

type Envs struct {
	arr []env
}

func (envs *Envs) Add(name, value string) {
	envs.arr = append(envs.arr, env{name, value})
}

func (envs *Envs) envVars() []corev1.EnvVar {
	if envs == nil {
		return nil
	}

	var rs []corev1.EnvVar
	for _, e := range envs.arr {
		rs = append(rs, corev1.EnvVar{Name: e.name, Value: e.value})
	}
	return rs
}
