# simple-operator
这是一个简单的Kubernetes Operator，部署 https://github.com/DAYUCS/simple-service 中的Spring Boot应用，并通过Kubernetes Service
发布出来。

## 环境准备
由于Operator SDK仅支持Linux、与Mac平台，开发、测试在Ubuntu上进行。准备开发环境时，除官方文档中所指明的必要条件外，还需安装GCC相关基本
软件：
```
sudo apt update
sudo apt install build-essential
```

## 参照
开发过程：https://courses.cognitiveclass.ai/courses/course-v1:IBM+CO0302EN+v1/course/ ，

程序说明：https://developer.ibm.com/learningpaths/kubernetes-operators/develop-deploy-simple-operator/deep-dive-memcached-operator-code/

（注意：以上两个参照未包含Kubernetes Service部分）

## 构建、部署Operator:
```
git clone https://github.com/DAYUCS/simple-operator.git
cd simple-operator
make deploy IMG=docker.io/biandayu/simple-operator:v1.0.10
```

## Operator部署成功后，请用以下命令部署应用：
```
kubectl apply -f config/samples/simple_v1alpha1_simple.yaml
```

## 应用部署成功后，请用以下命令得到应用的url:
```
minikube service --url simple-sample-service
```