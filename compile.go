package main

import (
	"context"
	"fmt"
	"time"

	//"context"
	//"fmt"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/krusty"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/api/konfig"
	"strings"
	//"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
)



func Compile(fs filesys.FileSystem, dirPath string) (resmap.ResMap, error) {
	buildOptions := &krusty.Options{
		UseKyaml:               false,
		DoLegacyResourceSort:   true,
		LoadRestrictions:       kustypes.LoadRestrictionsNone,
		AddManagedbyLabel:      false,
		DoPrune:                false,
		PluginConfig:           konfig.DisabledPluginConfig(),
		AllowResourceIdChanges: false,
	}

	k := krusty.MakeKustomizer(fs, buildOptions)
	return k.Run(dirPath)
}

func parseApplyOutput(in []byte) map[string]string {
	result := make(map[string]string)
	input := strings.Split(string(in), "\n")
	if len(input) == 0 {
		return result
	}
	var parts []string
	for _, str := range input {
		if str != "" {
			parts = append(parts, str)
		}
	}
	for _, str := range parts {
		kv := strings.Split(str, " ")
		if len(kv) > 1 {
			result[kv[0]] = kv[1]
		}
	}
	return result
}



func CheckDeployByKustomization(client dynamic.Interface, kustomization kustomizev1.Kustomization)  error{

	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{
		fmt.Sprintf("%s/name", kustomizev1.GroupVersion.Group):      kustomization.GetName(),
		fmt.Sprintf("%s/namespace", kustomizev1.GroupVersion.Group):  kustomization.GetNamespace(),
	}}
	glog.Infof("label is %s", labels.Set(labelSelector.MatchLabels).String())

	// read status snapshot
	if !kustomization.Spec.Prune || kustomization.Status.Snapshot == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*120))
	defer cancel()

	for ns, gvks := range kustomization.Status.Snapshot.NamespacedKinds() {
		for _, gvk := range gvks {
			glog.Infof("namespace is %s, group is %s, version is %s, resource is %s", ns, gvk.Group, gvk.Version, gvk.Kind)
			glog.Infof("label string is %s", labels.Set(labelSelector.MatchLabels).String())

			gvr := schema.GroupVersionResource{
				Group: gvk.Group,
				Version: gvk.Version,
				Resource: fmt.Sprint(strings.ToLower(gvk.Kind),"s"),
			}
			resourceList, err := client.Resource(gvr).Namespace(ns).List(ctx, metav1.ListOptions{
				LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
				Limit:         100,
			})
			if err == nil {
				for _, item := range resourceList.Items {
					id := fmt.Sprintf("%s/%s/%s", item.GetKind(), item.GetNamespace(), item.GetName())
					glog.Infof("resource is %s", id)
				}
			} else {
				glog.Infof("client query failed for %s: %v", gvk.Kind, err)
			}

			}
	}


	for _, gvk := range kustomization.Status.Snapshot.NonNamespacedKinds() {
			gvr := schema.GroupVersionResource{
				Group: gvk.Group,
				Version: gvk.Version,
				Resource: fmt.Sprint(strings.ToLower(gvk.Kind),"s"),
			}
			resourceList, err := client.Resource(gvr).List(ctx, metav1.ListOptions{
				LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
				Limit:         100,
			})
			if err == nil {
				for _, item := range resourceList.Items {
					id := fmt.Sprintf("%s/%s", item.GetKind(), item.GetName())
					glog.Infof("resource is %s", id)
				}
			} else {
				glog.Infof("client query failed for %s: %v", gvk.Kind, err)
			}


	}
	return nil
	// get the resource with label, if the resource name not in the channel, then output as delete

}