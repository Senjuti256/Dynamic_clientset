/*
Copyright 2023 Senjuti

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	
	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}


	// Register the Tekton scheme
	//v1beta1.AddToScheme(scheme.Scheme)
	//metav1.AddToGroupVersion(scheme.Scheme, schema.GroupVersion{Group: "tekton.dev", Version: "v1beta1"})


	// Create Task
	taskResource := schema.GroupVersionResource{Group: "tekton.dev", Version: "v1beta1", Resource: "tasks"}

	task := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "tekton.dev/v1beta1",
			"kind":       "Task",
			"metadata": map[string]interface{}{
				"name": "sample-task",
			},
			"spec": map[string]interface{}{
				"steps": []map[string]interface{}{
					{
						"name":  "step1",
        		"image": "ubuntu",
        		"command": []string{"echo", "Hello, Tekton!"},

					},
				},
			},
		},
	}

	fmt.Println("Creating Task...")
	resultTask, err := dynamicClient.Resource(taskResource).Namespace("default").Create(context.TODO(), task, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created Task %q.\n", resultTask.GetName())
	printTaskDetails(resultTask)


	// List Tasks
	prompt()
	fmt.Printf("Listing Tasks in namespace %q:\n", "default")
	taskList, err := dynamicClient.Resource(taskResource).Namespace("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, t := range taskList.Items {
		fmt.Printf(" * %s\n", t.GetName())
	}

	// Delete Task
	prompt()
	fmt.Println("Deleting Task...")
	deletePolicy := metav1.DeletePropagationForeground
	if err := dynamicClient.Resource(taskResource).Namespace("default").Delete(context.TODO(), "sample-task", metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
	}
	fmt.Println("Deleted Task.")
}

func prompt() {
	fmt.Printf("-> Press Return key to continue.")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	fmt.Println()
}

//func printTaskDetails(task *unstructured.Unstructured) {
	//image, _, _ := unstructured.NestedString(task.Object, "spec", "steps", "image")
	//command, _, _ := unstructured.NestedStringSlice(task.Object, "spec", "steps", "command")
	//fmt.Printf("Image: %s, Command: %v\n", image, command)
//}

func printTaskDetails(task *unstructured.Unstructured) {
	steps, found, err := unstructured.NestedSlice(task.Object, "spec", "steps")
	if err != nil || !found {
		fmt.Println("Error retrieving task details.")
		return
	}

	fmt.Println("Steps:")
	for i, step := range steps {
		stepMap, ok := step.(map[string]interface{})
		if !ok {
			fmt.Printf("Error processing step %d\n", i)
			continue
		}

		name, found, _ := unstructured.NestedString(stepMap, "name")
		if !found {
			name = fmt.Sprintf("Step%d", i+1)
		}

		image, found, _ := unstructured.NestedString(stepMap, "image")
		if !found {
			image = "N/A"
		}

		command, found, _ := unstructured.NestedStringSlice(stepMap, "command")
		if !found {
			command = []string{"N/A"}
		}

		fmt.Printf("  Step %d:\n", i+1)
		fmt.Printf("    Name: %s\n", name)
		fmt.Printf("    Image: %s\n", image)
		fmt.Printf("    Command: %v\n", command)
	}
}
