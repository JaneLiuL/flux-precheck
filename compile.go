package main

import (
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/krusty"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/api/konfig"
	"strings"

	//"github.com/golang/glog"
)


//
//func Compile(fs filesys.FileSystem, dirPath string)  ([]ResourceOutput, error){
////	kustomize build the manifest
//	_, err := buildKustomization(fs, dirPath)
//	if err != nil {
//		glog.Fatalf("Compile failed: %s", err)
//		return nil, err
//	}
//
////	kubectl dry run
//	return nil, err
//}


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