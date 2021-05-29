package main

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	//"fmt"
	flag "github.com/spf13/pflag"
	"github.com/golang/glog"
	"sigs.k8s.io/kustomize/api/filesys"
)

type ResourceOutput struct {
	action string
	kind string
	name string
}

const outputManifestFile = "flux-precheck-output-manifest.yaml"

func main()  {
//	flag to read folder, kustomization object
	var (
		manifestFolder string
		kustomizationName string
		kustomizationNamespace string
	)
	flag.StringVar(&manifestFolder, "manifest folder", "./", "The manifest folder")
	flag.StringVar(&kustomizationName, "kustomization object name", "./", "The kustomization object name")
	flag.StringVar(&kustomizationNamespace, "kustomization object namespace", "./", "The kustomization object namespace")
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
	applyCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := fmt.Sprintf("cd %s && kubectl apply -f %s --dry-run=server",outputManifestFile)
	command := exec.CommandContext(applyCtx, "/bin/sh", "-c", cmd)

	output, err := command.CombinedOutput()
	if err != nil {
		glog.Fatalf("Dry run failed: %w", err)
	}

	outputresources := parseApplyOutput(output)
	glog.Infof(
		fmt.Sprintf("Kustomization applied in %s",
			time.Now().Sub(start).String()),
		"output", outputresources,
	)
// read kustomization object name and namespace, read the status


//	get the resource with label, if the resource name not in the channel, then output as delete

// send to slack for the output

}
