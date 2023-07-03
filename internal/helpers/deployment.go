// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AnnotationPreDeleteHookStarted       = "vso.secrets.hashicorp.com/pre-delete-hook-started"
	AnnotationInMemoryVaultTokensRevoked = "vso.secrets.hashicorp.com/in-memory-vault-tokens-revoked"
	LabelSelectorControlPlane            = "control-plane=controller-manager"
	StringTrue                           = "true"
)

func AwaitInMemoryVaultTokensRevoked(ctx context.Context, logger logr.Logger, c client.Client) {
	selector, err := labels.Parse(LabelSelectorControlPlane)
	if err != nil {
		logger.Error(err, "failed to parse label selector", "selector", LabelSelectorControlPlane)
		return
	}

	for {
		select {
		case <-ctx.Done():
			logger.Error(ctx.Err(), "failed to await in-memory vault tokens revoked")
			return
		default:
			var list corev1.PodList
			err = c.List(ctx, &list, client.MatchingLabelsSelector{
				Selector: selector,
			})
			if err != nil {
				logger.Error(err, "failed to get pod list", "selector", LabelSelectorControlPlane)
			} else {
				for _, pod := range list.Items {
					if value, ok := pod.Annotations[AnnotationInMemoryVaultTokensRevoked]; ok && value == StringTrue {
						logger.Info("Operator pods annotations updated", AnnotationInMemoryVaultTokensRevoked, StringTrue)
						return
					}
				}
			}
			time.Sleep(300 * time.Millisecond)
		}
	}
}

func AwaitPreDeleteStarted(ctx context.Context, handler context.CancelFunc, logger logr.Logger) {
	for {
		select {
		case <-ctx.Done():
			logger.Error(ctx.Err(), "Operator manager context canceled. Stopping /var/run/podinfo/pre-delete-hook-started watcher")
			return
		default:
			if b, err := os.ReadFile("/var/run/podinfo/pre-delete-hook-started"); err != nil {
				logger.Error(err, "failed to get downward API exposed file", "path", "/var/run/podinfo/pre-delete-hook-started")
			} else if string(b) == StringTrue {
				logger.Info("Operator pods annotations updated", AnnotationPreDeleteHookStarted, StringTrue)
				handler()
				return
			}
			time.Sleep(300 * time.Millisecond)
		}
	}
}

func AnnotateInMemoryVaultTokensRevoked(ctx context.Context, c client.Client) error {
	return annotateOperatorPods(ctx, c, map[string]string{AnnotationInMemoryVaultTokensRevoked: StringTrue})
}

func AnnotatePredeleteHookStarted(ctx context.Context, c client.Client) error {
	return annotateOperatorPods(ctx, c, map[string]string{AnnotationPreDeleteHookStarted: StringTrue})
}

func annotateOperatorPods(ctx context.Context, c client.Client, annotations map[string]string) error {
	var list corev1.PodList

	selector, err := labels.Parse(LabelSelectorControlPlane)
	if err != nil {
		return fmt.Errorf("failed to parse label selector err=%v", err)
	}

	err = c.List(ctx, &list, client.MatchingLabelsSelector{
		Selector: selector,
	})
	if err != nil {
		return fmt.Errorf("failed to list pods err=%v", err)
	}

	errs := []string{}
	for _, pod := range list.Items {
		for k, v := range annotations {
			pod.Annotations[k] = v
		}
		pJson, err := json.Marshal(pod)
		if err != nil {
			return fmt.Errorf("failed to marshal patch payload err=%v", err)
		}
		if err = c.Patch(ctx, &pod, client.RawPatch(types.MergePatchType, pJson)); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) != 0 {
		return fmt.Errorf(strings.Join(errs, ","))
	}
	return nil
}
