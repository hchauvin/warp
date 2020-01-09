// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

package env

import (
	"github.com/hchauvin/warp/pkg/stacks/names"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestMatchConfigMap(t *testing.T) {
	stackName := names.Name{Family: "stack", ShortName: "0"}

	list := &corev1.ConfigMapList{
		Items: []corev1.ConfigMap{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "stack-0-foo",
				},
			},
		},
	}
	cfgmap, err := matchConfigMap(list, stackName, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "stack-0-foo", cfgmap.Name)

	list = &corev1.ConfigMapList{
		Items: []corev1.ConfigMap{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "stack-0-foo-randompart",
				},
			},
		},
	}
	cfgmap, err = matchConfigMap(list, stackName, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "stack-0-foo-randompart", cfgmap.Name)

	list = &corev1.ConfigMapList{}
	cfgmap, err = matchConfigMap(list, stackName, "foo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no matching config map found")

	list = &corev1.ConfigMapList{
		Items: []corev1.ConfigMap{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "stack-0-foo",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "stack-0-foo-randompart",
				},
			},
		},
	}
	cfgmap, err = matchConfigMap(list, stackName, "foo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple matching config maps found")
}

func TestConfigMapEntryValue(t *testing.T) {
	data := map[string]string{
		"foo": "bar",
		"qux": "wobble",
	}
	v, err := configMapEntryValue("resourceName", data, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "bar", v)

	v, err = configMapEntryValue("resourceName", data, "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key 'unknown' was not found in config map resourceName")
}

func TestMatchSecret(t *testing.T) {
	stackName := names.Name{Family: "stack", ShortName: "0"}

	list := &corev1.SecretList{
		Items: []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "stack-0-foo",
				},
			},
		},
	}
	cfgmap, err := matchSecret(list, stackName, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "stack-0-foo", cfgmap.Name)

	list = &corev1.SecretList{
		Items: []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "stack-0-foo-randompart",
				},
			},
		},
	}
	cfgmap, err = matchSecret(list, stackName, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "stack-0-foo-randompart", cfgmap.Name)

	list = &corev1.SecretList{}
	cfgmap, err = matchSecret(list, stackName, "foo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no matching secret found")

	list = &corev1.SecretList{
		Items: []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "stack-0-foo",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "stack-0-foo-randompart",
				},
			},
		},
	}
	cfgmap, err = matchSecret(list, stackName, "foo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple matching secrets found")
}

func TestSecretEntryValue(t *testing.T) {
	data := map[string][]byte{
		"foo": []byte("bar"),
		"qux": []byte("wobble"),
	}
	v, err := secretEntryValue("resourceName", data, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "bar", v)

	v, err = secretEntryValue("resourceName", data, "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key 'unknown' was not found in secret resourceName")
}
