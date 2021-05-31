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
	//"github.com/golang/glog"
	"github.com/gogf/gf/os/glog"
	"sigs.k8s.io/kustomize/api/filesys"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/dynamic"
)


const outputManifestFile = "flux-precheck-output-manifest.yaml"


var (
	manifestFolder string
	kustomizationName string
	kustomizationNamespace string
	kubeconfigPath string
	clientset *kubernetes.Clientset
	data []byte
)

func  init()  {
	flag.StringVar(&manifestFolder, "manifestFolder", "./", "The manifest folder")
	flag.StringVar(&kustomizationName, "kustomizationName", "kustomization-name", "The kustomization object name")
	flag.StringVar(&kustomizationNamespace, "kustomizationNamespace", "kustomization-namespace", "The kustomization object namespace")
	flag.StringVar(&kubeconfigPath, "kubeconfigPath", "/root/.kube/config", "The kube config file path")
	flag.Parse()
}

func main()  {
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

	command := exec.CommandContext(applyCtx, "/bin/sh", "-c", cmd)

	output, err := command.CombinedOutput()
	if err != nil {
		glog.Fatalf("Dry run failed: %w", err)
	}

	outputresources := parseApplyOutput(output)
	glog.Infof(fmt.Sprintf("dry run output %s", outputresources))
	//output format: map[Warning::kubectl deployment.apps/my-dep:configured pod/test:created]

	resourceoutput := make(map[string]string)
	for obj, action := range outputresources {

		if obj == "Warning:" {
			glog.Info("Skip warning")
			continue
		}
		resourceoutput[fmt.Sprintf("%s/%s", strings.Split(obj, "/")[0],strings.Split(obj, "/")[1])] = action

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
	lastTimekustomizationDeployOutput, err := CheckDeployByKustomization(client, kustomization)


// As we already know what will be  created / unchanged / configured by dry-run
// we need to compare with CheckDeployByKustomization output, to know what will be deleted


	for _, v := range lastTimekustomizationDeployOutput {
		ss := strings.Split(v, "/")


		if _, found := resourceoutput[fmt.Sprintf("%s/%s", ss[0], ss[len(ss)-1])] ; found {
			fmt.Println(v)
		} else {
		//	this resource will be delete
			resourceoutput[fmt.Sprintf("%s/%s", ss[0],ss[len(ss)-1])] = "deleted"
		}
	}

	glog.Info("============result==============")

	for mapk, mapv := range resourceoutput {
		glog.Infof("%s action is %s", mapk, mapv)
	}
}
