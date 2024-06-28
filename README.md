# k8s-svc-keepalive-timeout-scanner
scan service keepalive timeout in k8s 

# Background
If the service use keepalive and set the keepalive-idle-timeout is shorter the the client keepalive-idle-timeout,sometimes the request will reset because the server active close the  connection after timeout.
So we need to make sure the client keepalive-idle-timeout is shorter than it in server.
But the problem is that is not easy to know all the services in k8s cluster, because too many services!
This script is use to scan the service keepalive-idle-timeout in k8s cluster

在长链接设置中，我们需要确认服务端的长链接空闲超时时间比客户端的要短，确认服务端不会主动关闭长链接。
否则如果服务端主动关闭发出 fin 包之后，客户端这时还有一个请求正在传输的话，这个请求将会被 reset 掉。
但是在 k8s 集群中，部署了非常多的服务，我们很难一个服务一个服务地去看配置获取长链接的配置，所以这个工具的作用就是对服务发出一个请求，然后等待长链接超时之后由服务端主动关闭链接，来计算出服务的长链接空闲超时时间


# 原理
1. 这个脚本编译之后需要在 k8s 集群的 pod 中运行，因为这个脚本需要访问每个 pod
2. 需要 get list deployment、get list pod 的权限
3. 脚本先获取指定 namespace 下的所有 deployment，然后获取 deployment 的第一个 pod 的信息，获取 readiness probe 信息和 pod ip
4. 请求一次 pod 的 readiness，记下开始时间，然后等待链接关闭，记下结束时间，用结束时间减去开始时间就可以知道 server 的空闲超时时间
5. 如果等待 100s 还没有断开的，这个脚本就主动断开

# Usage
1. 先编译
   ```azure
    go build
   ```
2. 上传到一个 pod 作为 client，这个 pod 使用的 serviceaccount 必须有 get、list deployment 和 pod 的权限
3. 执行
   ```azure
   ./scanner --ns <namespace>
   ```