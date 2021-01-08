package secrets

import (
	"github.com/nais/azureator/pkg/azure"
	corev1 "k8s.io/api/core/v1"
)

type Lists struct {
	Used   corev1.SecretList
	Unused corev1.SecretList
}

func podSecretLists(secrets corev1.SecretList, pods corev1.PodList) Lists {
	lists := Lists{
		Used: corev1.SecretList{
			Items: make([]corev1.Secret, 0),
		},
		Unused: corev1.SecretList{
			Items: make([]corev1.Secret, 0),
		},
	}

	for _, sec := range secrets.Items {
		if secretInPods(sec, pods) {
			lists.Used.Items = append(lists.Used.Items, sec)
		} else {
			lists.Unused.Items = append(lists.Unused.Items, sec)
		}
	}
	return lists
}

func secretInPod(secret corev1.Secret, pod corev1.Pod) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.Secret != nil && volume.Secret.SecretName == secret.Name {
			return true
		}
	}
	for _, container := range pod.Spec.Containers {
		for _, envFrom := range container.EnvFrom {
			if envFrom.SecretRef != nil && envFrom.SecretRef.Name == secret.Name {
				return true
			}
		}
	}
	return false
}

func secretInPods(secret corev1.Secret, pods corev1.PodList) bool {
	for _, pod := range pods.Items {
		if secretInPod(secret, pod) {
			return true
		}
	}
	return false
}

func WithIdsFromUsedSecrets(a azure.ApplicationResult, s Lists) azure.ApplicationResult {
	passwordIds := make([]string, 0)
	certificateIds := make([]string, 0)
	for _, sec := range s.Used.Items {
		certificateId := string(sec.Data[CertificateIdKey])
		if len(certificateId) > 0 {
			certificateIds = append(certificateIds, certificateId)
		}
		passwordId := string(sec.Data[PasswordIdKey])
		if len(passwordId) > 0 {
			passwordIds = append(passwordIds, passwordId)
		}
	}
	a.Password.KeyId.AllInUse = passwordIds
	a.Certificate.KeyId.AllInUse = certificateIds
	return a
}
