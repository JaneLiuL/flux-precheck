package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/golang/glog"
	"sigs.k8s.io/kustomize/api/filesys"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/dynamic"
)

type ResourceOutput struct {
	action string
	kind string
	name string
}

const outputManifestFile = "flux-precheck-output-manifest.yaml"

func main()  {

	var (
		manifestFolder string
		kustomizationName string
		kustomizationNamespace string
		kubeconfigPath string
		clientset *kubernetes.Clientset
		data []byte
	)
	flag.StringVar(&manifestFolder, "manifest folder", "./", "The manifest folder")
	flag.StringVar(&kustomizationName, "kustomization object name", "kustomization-name", "The kustomization object name")
	flag.StringVar(&kustomizationNamespace, "kustomization object namespace", "default", "The kustomization object namespace")
	flag.StringVar(&kubeconfigPath, "Kube config file path", "/root/.kube/config", "The kube config file path")
	flag.Parse()

//	compile and dry run apply and output the create and configured
	fs := filesys.MakeFsOnDisk()
	m, err := Compile(fs, manifestFolder)
	if err != nil {
		glog.Fatalf("Compile failed: %s", err)
	}

	resources, err := m.AsYaml()
	if err != nil {
		glog.Fatalf("kustomize build failed: %w", err)
	}

	manifestsFile := filepath.Join(manifestFolder, outputManifestFile)

	if err := fs.WriteFile(manifestsFile, resources); err != nil {
		glog.Fatalf("Writing manifest failed: %w", err)
	}

//	dry run and get the output
	applyCtx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*120))
	defer cancel()

	cmd := fmt.Sprintf("kubectl apply -f %s --dry-run=client", manifestsFile)
	glog.Infof("command output as %s",cmd)
	command := exec.CommandContext(applyCtx, "/bin/sh", "-c", cmd)

	output, err := command.CombinedOutput()
	if err != nil {
		glog.Fatalf("Dry run failed: %w", err)
	}

	outputresources := parseApplyOutput(output)
	glog.Infof(fmt.Sprintf("dry run output %s", outputresources))
	//output format: map[Warning::kubectl deployment.apps/my-dep:configured pod/test:created]
	var resourceoutput []ResourceOutput

	for obj, action := range outputresources {

		if obj == "Warning:" {
			glog.Info("Skip warning")
			continue
		}
		resourceoutput = append(resourceoutput, ResourceOutput{
			action: action,
			kind: strings.Split(obj, "/")[0],
			name: strings.Split(obj, "/")[1],
		})
	}

// read kustomization object name and namespace, read the status
	config, err := clientcmd.BuildConfigFromFlags("",kubeconfigPath)
	if err != nil {
		glog.Fatalf("Client build config fail: %w", err)
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	data, err = clientset.RESTClient().
		Get().
		AbsPath("/apis/kustomize.toolkit.fluxcd.io/v1beta1").
		Namespace(kustomizationNamespace).
		Resource("kustomizations").
		Name(kustomizationName).
		DoRaw(context.TODO())

	if len(data) == 0 {
		glog.Fatal("Get resource fail")
	}

	//glog.Infof("output data is %s", string(data))
	var kustomization kustomizev1.Kustomization

	if err := json.Unmarshal(data, &kustomization); err != nil {
		glog.Fatalf("Unmarshal fail: %w", err)
	}

	client, err := dynamic.NewForConfig(config)
	_ = CheckDeployByKustomization(client, kustomization)



}
