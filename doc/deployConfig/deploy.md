1、创建一个namespace
kubectl apply - f namespace.yaml
2、在该namespace创建limit和resourcequota
kubectl apply -f limits.yaml
kubectl apply -f compute-resource.yaml
3、创建deployment使用resource
kubectl apply -f deployment.yaml
4、查看resourcequota使用
kubectl get resourcequota -nquota-example
5、部署控制器
make docker-build
// 修改dockerFile、makefile
// kind 需要loadimage
kind load docker-image dynamicquota:v5 --name k8s1.18.3node
# kind load docker-image dynamicquota:v2 --name k8s1.16.3node
make deploy
// make deploy会将controller部署到一个新的namespace中，[项目名称-system]，default/kustomization.yaml中修改namespace
kubectl get all // 查看controler manager
kubectl get crd // 查看crd
6、如果没有权限
kubectl apply -f mysa.yaml
