package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/fatih/color"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"net"
	"sync"
	"time"
)

var (
	red = color.New(color.FgRed).SprintFunc()
	wg  sync.WaitGroup
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "kuebconfig path")
	namespace := flag.String("n", "", "namespace")
	flag.Parse()
	// 创建 Kubernetes 客户端配置
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(any(err))
	}

	// 创建 Kubernetes 客户端
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(any(err))
	}

	// 获取指定 namespace 中的所有 pod
	//podList, err := clientset.CoreV1().Pods(*namespace).List(context.Background(), metav1.ListOptions{})
	// 获取指定 namespace 中的所有 deployment
	deploymentList, err := clientset.AppsV1().Deployments(*namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(any(err))
	}
	i := 1
	for _, deployment := range deploymentList.Items {
		i++
		deploymentName := deployment.Name
		labelSelector := metav1.LabelSelector{MatchLabels: deployment.Spec.Selector.MatchLabels}
		options := metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(&labelSelector)}
		podList, _ := clientset.CoreV1().Pods(*namespace).List(context.Background(), options)
		// 获取 pod 的 readiness 配置信息
		for _, pod := range podList.Items {
			for _, container := range pod.Spec.Containers {
				i++
				//containerName := container.Name
				readinessProbe := container.ReadinessProbe
				if readinessProbe == nil {
					continue
				}
				if readinessProbe.HTTPGet != nil {
					port := readinessProbe.HTTPGet.Port.IntVal
					//scheme := readinessProbe.HTTPGet.Scheme
					path := readinessProbe.HTTPGet.Path
					podIp := pod.Status.PodIP
					address := fmt.Sprintf("%v:%v", podIp, port)
					//url := fmt.Sprintf("%v://%v:%v%v", scheme, podIp, port, path)
					wg.Add(i)
					go request("GET", address, path, deploymentName)
				}
			}
			// 只拿一个就行了
			break
		}
	}
	wg.Wait()
}

func request(method string, address string, path string, deploymentName string) {
	//fmt.Println(deploymentName)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}

	defer conn.Close()
	defer wg.Done()
	// 构建 HTTP GET 请求报文
	req := "GET " + path + " HTTP/1.1\r\n" +
		"Host: " + address + "\r\n" +
		"Connection: keep-alive\r\n\r\n"
	fmt.Fprintf(conn, req)
	// 读取响应报文
	reader := bufio.NewReader(conn)
	start := time.Now()
	startTimestamp := start.UnixNano() / int64(time.Millisecond) / 1000
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading:", err)
			return
		}
		//fmt.Println(line)
		if line == "\r\n" {
			break
		}
	}
	timeout := 100 * time.Second
	conn.SetDeadline(time.Now().Add(timeout))
	for {
		buf := make([]byte, 1024)
		n, err := reader.Read(buf)
		if err != nil {
			end := time.Now()
			endTimestamp := end.UnixNano() / int64(time.Millisecond) / 1000
			timeout := endTimestamp - startTimestamp
			fmt.Printf("%v: %v\n", deploymentName, timeout)
			return
		}
		if n == 0 {
			break
		}
		//fmt.Print(string(buf[:n]))
	}

}
