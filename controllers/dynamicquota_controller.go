/*


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

package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	re "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"math"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strconv"
	"strings"

	webappv1 "dynamicQuota/api/v1"
)

// DynamicQuotaReconciler reconciles a DynamicQuota object
type DynamicQuotaReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=webapp.zt.domain,resources=dynamicquota,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.zt.domain,resources=dynamicquota/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=resourcequotas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pod,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pod/status,verbs=get;
// +kubebuilder:rbac:groups=core,resources=resourcequotas/status,verbs=get

func (r *DynamicQuotaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("dynamicQuota", req.NamespacedName)
	//log.V(1).Info("start -----------")
	fmt.Println("start -----------")
	fmt.Println("reqBy：---" + req.Name)
	log.V(1).Info("req info", "Namespace", req.Namespace, "Name", req.Name)
	var dynamicQuotaList webappv1.DynamicQuotaList
	var dynamicQuota webappv1.DynamicQuota
	var allNameSpaces corev1.NamespaceList
	var nameSpaceList []string

	// 获取指定 node 的信息
	getNodeInfo := func(node *corev1.Node) (result map[string]int32, err error) {
		result = make(map[string]int32, 2)
		CPU := node.Status.Capacity["cpu"]
		MEM := node.Status.Capacity["memory"]
		cpu, _ := CPU.AsInt64()
		mem, _ := MEM.AsInt64()
		result["cpu"] = int32(cpu)
		result["mem"] = int32(mem / 1024 / 1024) // 内存以M为单位
		return result, nil
	}

	// 获取当前集群的节点信息
	getClusterNodeInfo := func() (cpu int32, mem int32, nodes []string, err error) {
		//获取所有的node，计算totalCpu、totalMem、totalNet
		var nodeList corev1.NodeList
		if err := r.List(ctx, &nodeList); err != nil {
			log.Error(err, "unable to list node")
			return cpu, mem, nodes, err
		}
		for _, node := range nodeList.Items {
			nodeInfo, err := getNodeInfo(&node)
			if err != nil {
				log.Error(err, "unable to get nodeInfo of "+node.Name)
				return cpu, mem, nodes, err
			}
			cpu = nodeInfo["cpu"] + cpu
			mem = nodeInfo["mem"] + mem
			nodes = append(nodes, node.Name)
		}
		return cpu, mem, nodes, nil
	}

	// 获取dynamicQuota对象，没有则返回
	if err := r.List(ctx, &dynamicQuotaList); err != nil {
		log.Error(err, "unable to fetech DynamicQuota")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	} else {
		if len(dynamicQuotaList.Items) > 0 {
			dynamicQuota = dynamicQuotaList.Items[0]
			// 如果本次是dynamicQuota的请求，则初始化TotalCpu等信息
			if req.Name == dynamicQuota.Name {
				// 获取所有的ns
				if err := r.List(ctx, &allNameSpaces); err != nil {
					log.Error(err, "unable to list nameSpaceList")
					return ctrl.Result{}, err
				}

				// 如果没有计算过资源数据，第一次创建DynamicQuota对象，初始化数据
				if dynamicQuota.Status.TotalCpu == nil || dynamicQuota.Status.TotalMem == nil {
					// 初始化
					dynamicQuota.Status.TotalCpu = new(int32)
					dynamicQuota.Status.TotalMem = new(int32)
					dynamicQuota.Status.TotalNet = new(int32)
					cpu, mem, nodes, err := getClusterNodeInfo()
					if err != nil {
						return ctrl.Result{}, nil
					}
					dynamicQuota.Status.TotalCpu = &cpu
					dynamicQuota.Status.TotalMem = &mem
					dynamicQuota.Status.NodeList = nodes
					log.V(1).Info("cluster info", "totalCpu", dynamicQuota.Status.TotalCpu, "totalMem", dynamicQuota.Status.TotalMem, "nodeList", dynamicQuota.Status.NodeList)
					if err := r.Status().Update(ctx, &dynamicQuota); err != nil {
						log.Error(err, "unable to update DynamicQuota status")
						return ctrl.Result{}, err
					}
					// ns to map
					allNameSpaceList := make(map[string]bool, 5)
					for _, ns := range allNameSpaces.Items {
						allNameSpaceList[ns.Name] = true
					}
					// 第一次创建时检查，获取指定的namespace列表，验证是否存在该namespace
					nameSpaceList = dynamicQuota.Spec.NameSpaces
					for _, ns := range nameSpaceList {
						if !allNameSpaceList[ns] {
							log.Error(nil, "dynamicQuota.Spec.NameSpaces has an invalid value:"+ns)
						}
					}
				}

			}
		} else {
			fmt.Println("还没有创建DynamicQuota")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	// 计算变化比率，增加节点是增长率，删除节点是减少率，修改resourceQuota
	changeResourceByNodeInfo := func(dq *webappv1.DynamicQuota) error {
		var ratio float64
		var oldCpu = *dq.Status.TotalCpu
		var oldMem = *dq.Status.TotalMem
		var cpuRatio float64
		var memRatio float64

		newCpu, newMem, newNodes, err := getClusterNodeInfo()

		if err != nil {
			return nil
		}
		if len(newNodes) > len(dynamicQuota.Status.NodeList) {
			// 增加节点，计算增长率
			cpuRatio = float64(newCpu) / float64(oldCpu)
			memRatio = float64(newMem) / float64(oldMem)
		} else if len(newNodes) < len(dynamicQuota.Status.NodeList) {
			// 删除节点，计算减少率
			cpuRatio = float64(newCpu) / float64(oldCpu)
			memRatio = float64(newMem) / float64(oldMem)
		} else {
			fmt.Printf("节点数量没有变化!")
		}

		if dq.Spec.ChangePolicy == webappv1.MeanPolicy {
			ratio = (cpuRatio + memRatio) / 2
		} else if dq.Spec.ChangePolicy == webappv1.MaxPolicy {
			ratio = math.Max(cpuRatio, memRatio)
		} else {
			ratio = math.Min(cpuRatio, memRatio)
		}
		log.V(1).Info("ratio compute", "cpuRatio", cpuRatio, "memRatio", memRatio)
		var resourceQuota corev1.ResourceQuota
		var resourceQuotaList corev1.ResourceQuotaList
		// 得到增长率后进行resourceQuota的改变
		for _, ns := range dq.Spec.NameSpaces {
			if err := r.List(ctx, &resourceQuotaList, client.InNamespace(ns)); err != nil {
				log.Error(err, "unable to list resourceQuota", "namespace", ns)
			}
			if len(resourceQuotaList.Items) < 1 {
				log.V(1).Info("该namespace下没有resourceQuota", "namespace", ns)
			} else {
				resourceQuota = resourceQuotaList.Items[0]

				podNumber := resourceQuota.Spec.Hard["pods"]
				pn, _ := podNumber.AsInt64()
				resourceQuota.Spec.Hard["pods"] = re.MustParse(strconv.Itoa(int(float64(pn) * ratio)))

				limitCpu := resourceQuota.Spec.Hard["limits.cpu"]
				lc := limitCpu.String()
				var lcI float64
				if !strings.Contains(lc, "m") {
					lc2, _ := strconv.Atoi(lc)
					lcI = float64(lc2 * 1000)
				} else {
					lcI, _ = strconv.ParseFloat(strings.Trim(lc, "m"), 64)
				}
				resourceQuota.Spec.Hard["limits.cpu"] = re.MustParse(strconv.Itoa(int(lcI*ratio)) + "m")

				limitMem := resourceQuota.Spec.Hard["limits.memory"]
				lm, _ := limitMem.AsInt64()
				resourceQuota.Spec.Hard["limits.memory"] = re.MustParse(strconv.Itoa(int(float64(lm)*ratio)/1024/1024) + "Mi")

				requestCpu := resourceQuota.Spec.Hard["requests.cpu"]
				rc := requestCpu.String()
				var rcI float64
				if !strings.Contains(rc, "m") {
					rc2, _ := strconv.Atoi(rc)
					rcI = float64(rc2 * 1000)
				} else {
					rcI, _ = strconv.ParseFloat(strings.Trim(rc, "m"), 64)
				}
				resourceQuota.Spec.Hard["requests.cpu"] = re.MustParse(strconv.Itoa(int(rcI*ratio)) + "m")

				requestMem := resourceQuota.Spec.Hard["requests.memory"]
				rm, _ := requestMem.AsInt64()
				resourceQuota.Spec.Hard["requests.memory"] = re.MustParse(strconv.Itoa(int(float64(rm)*ratio)/1024/1024) + "Mi")

				fmt.Println(resourceQuota.Spec)
				log.V(1).Info("ratio data", "policy", dq.Spec.ChangePolicy, "ratio", ratio, "namespace", ns, "resourceQuota", resourceQuota.Name, "limit.cpu", lc, "limit.mem", lm, "request.cpu", rc, "request.mem", rm)
				if err := r.Update(ctx, &resourceQuota); err != nil {
					//if err := r.Status().Update(ctx, &resourceQuota); err != nil {
					log.Error(err, "unable to update resourceQuota status")
					return err
				} else {
					fmt.Println("update resourceQuota 成功")
				}
			}
		}
		return nil
	}

	if req.Namespace == "createNode" || req.Namespace == "deleteNode" {
		changeResourceByNodeInfo(&dynamicQuota)
	}


	newCpu, newMem, newNodes, err := getClusterNodeInfo()

	if err != nil {
		return ctrl.Result{}, err
	}
	dynamicQuota.Status.TotalCpu = &newCpu
	dynamicQuota.Status.TotalMem = &newMem
	dynamicQuota.Status.NodeList = newNodes

	log.V(1).Info("cluster info", "totalCpu", dynamicQuota.Status.TotalCpu, "totalMem", dynamicQuota.Status.TotalMem, "nodeList", dynamicQuota.Status.NodeList)
	if err := r.Status().Update(ctx, &dynamicQuota); err != nil {
		log.Error(err, "unable to update DynamicQuota status")
		return ctrl.Result{}, err
	}

	fmt.Println("ending----------")

	return ctrl.Result{}, nil
}

var (
	nodeOwnerKey = ".metadata.controller"
	apiGVStr     = webappv1.GroupVersion.String()
)

// 自定义事件触发Reconcile
type EnqueueRequestForNode struct{}

// Create implements EventHandler
func (e *EnqueueRequestForNode) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Object.GetName(),
		Namespace: "createNode",
		//Namespace: evt.Object.GetNamespace(),
	}})
}

func (e *EnqueueRequestForNode) Update(updateEvent event.UpdateEvent, q workqueue.RateLimitingInterface) {
	//klog.V(1).Info("update 节点事件，不触发更新逻辑")
	//fmt.Printf("update 节点事件，不触发更新逻辑")
}

func (e *EnqueueRequestForNode) Delete(deleteEvent event.DeleteEvent, q workqueue.RateLimitingInterface) {
	fmt.Printf("delete事件")
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      deleteEvent.Object.GetName(),
		Namespace: "deleteNode",
		//Namespace: deleteEvent.Object.GetNamespace(),
	}})
}

func (e *EnqueueRequestForNode) Generic(genericEvent event.GenericEvent, q workqueue.RateLimitingInterface) {
	//klog.V(1).Info("Generic 节点事件，不触发更新逻辑")
	//fmt.Printf("update 节点事件，不触发更新逻辑")
}

func (r *DynamicQuotaReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.DynamicQuota{}).
		Watches(&source.Kind{Type: &corev1.Node{}}, &EnqueueRequestForNode{}).
		Complete(r)
}
