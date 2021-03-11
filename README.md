# 一初始化
## 1、go mod init dynamicQuota
## 2、kubebuilder init --domain zt.domain
## 3、kubebuilder create api --group webapp --version v1 --kind DynamicQuota


# 二、make install
Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -
kustomize build 会将crd下的文件解析成  config/crd/bases/webapp.zt.domain_dynamicquota.yaml
然后kubectl apply -f webapp.zt.domain_dynamicquota.yaml创建 CRD

    // +kubebuilder:resource:scope=Cluster  # 表示集群的资源

# 三、reconcile调用
   每次更新我们的CRD资源才会触发reconcile方法，想要监控多个资源变化
   在SetupWithManager中使用for来监控，如下面监控Node的资源变化
    func (r *DynamicQuotaReconciler) SetupWithManager(mgr ctrl.Manager) error {
        return ctrl.NewControllerManagedBy(mgr).
            For(&webappv1.DynamicQuota{}).
            Watches(&source.Kind{Type: &corev1.Node{}}, &EnqueueRequestForNode{}).
            Complete(r)
    }
# 四、开发调试
## 1、运行
    make install && make run
    make uninstall && make install && make run 
## 2、启动一个crd实例
    kubectl apply -f config/samples
## 3、创建相关资源
	kubectl apply -f doc/deployconfig/
## 3、创建一个namespace触发resourcequota的扩容
    kubectl create ns
## 4、删除namespace触发resourcequota的缩容
    kubectl delete ns 
