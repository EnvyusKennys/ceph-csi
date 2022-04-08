package util

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func CheckIfReadonlyMount(po *v1.Pod) (bool, error) {
	for _, vol := range po.Spec.Volumes {
		if vol.PersistentVolumeClaim != nil {

			if !vol.PersistentVolumeClaim.ReadOnly {
				for _, con := range po.Spec.Containers {
					if con.VolumeMounts == nil {
						continue
					}
					for _, vm := range con.VolumeMounts {
						if vm.Name == vol.Name && vm.ReadOnly {
							return true, nil
						}
					}
				}
				return false, nil
			}
			return true, nil

		}
	}
	return false, fmt.Errorf("no matching conditions")
}

func GetPod(name string, namespace string) (*v1.Pod, error) {
	c := NewK8sClient()
	pod, err := c.CoreV1().Pods(namespace).Get(context.TODO(), name, metav1.GetOptions{})

	if err != nil {
		klog.V(6).Infof("Can't get pod %s namespace %s: %v", name, namespace, err)
		return nil, err
	}

	return pod, nil
}
