/*
Copyright 2023 Senjuti De

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

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}

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
						"name": "step1",
						"containers": map[string]interface{}{
							"image": "ubuntu",
							"command": []string{"echo", "Hello, Tekton!"},
						},
					},
				},
			},
		},
	}

	fmt.Println("Creating Task...")
	resultTask, err := dynamicClient.Resource(taskResource).Create(context.TODO(), task, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created Task %q.\n", resultTask.GetName())
	printTaskDetails(resultTask)

	// Update Task
	prompt()
	fmt.Println("Updating Task...")
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		resultTask, getErr := dynamicClient.Resource(taskResource).Get(context.TODO(), "sample-task", metav1.GetOptions{})
		if getErr != nil {
			panic(fmt.Errorf("Failed to get the latest version of Task: %v", getErr))
		}

		// Print task details before update
		fmt.Printf("Before Update - ")
		printTaskDetails(resultTask)

		// Modify the Task as needed
		// For example, change the image to busybox and update the message
		unstructured.SetNestedField(resultTask.Object, "busybox", "spec", "steps", "0", "container", "image")
		unstructured.SetNestedField(resultTask.Object, []string{"echo", "Updated Hello Tekton"}, "spec", "steps", "0", "container", "command")

		// Print task details after update
		fmt.Printf("After Update - ")
		printTaskDetails(resultTask)

		_, updateErr := dynamicClient.Resource(taskResource).Update(context.TODO(), resultTask, metav1.UpdateOptions{})
		return updateErr
	})
	if retryErr != nil {
		panic(fmt.Errorf("Update failed: %v", retryErr))
	}
	fmt.Println("Updated Task...")
	fmt.Printf("Updated Task %q. ", resultTask.GetName())
	printTaskDetails(resultTask)

	// List Tasks
	prompt()
	fmt.Printf("Listing Tasks in namespace %q:\n", "default")
	taskList, err := dynamicClient.Resource(taskResource).List(context.TODO(), metav1.ListOptions{})
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
	if err := dynamicClient.Resource(taskResource).Delete(context.TODO(), "sample-task", metav1.DeleteOptions{
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

func printTaskDetails(task *unstructured.Unstructured) {
	image, _, _ := unstructured.NestedString(task.Object, "spec", "steps", "0", "container", "image")
	command, _, _ := unstructured.NestedStringSlice(task.Object, "spec", "steps", "0", "container", "command")
	fmt.Printf("Image: %s, Command: %v\n", image, command)
}
