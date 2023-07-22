package k8s

import (
	"context"
	"fmt"
	"pggat2/lib/gat"

	"tuxpa.in/a/zlog/log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type PodWatcher struct {
	BaseRecipe gat.Recipe

	Namespace   string
	ListOptions metav1.ListOptions

	pods map[string]*v1.Pod
}

func (p *PodWatcher) Start(
	ctx context.Context,
	c *kubernetes.Clientset,
	pool gat.Pool,
) error {
	p.pods = make(map[string]*v1.Pod)
	err := p.getInitialPods(ctx, c, pool)
	if err != nil {
		return err
	}
	return p.startWatching(ctx, c, pool)
}

func (p *PodWatcher) getInitialPods(
	ctx context.Context,
	c *kubernetes.Clientset,
	pool gat.Pool,
) error {
	pods, err := c.CoreV1().Pods(p.Namespace).List(ctx, p.ListOptions)
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		if isPodReady(&pod) {
			p.pods[pod.Name] = &pod
		}
	}
	return nil
}

func (p *PodWatcher) startWatching(
	ctx context.Context,
	c *kubernetes.Clientset,
	pool gat.Pool,
) error {
	watcher, err := c.CoreV1().Pods(p.Namespace).Watch(ctx, p.ListOptions)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		pod, ok := event.Object.(*v1.Pod)
		if !ok {
			continue
		}

		podName := pod.Name
		podIp := pod.Status.PodIP
		podReady := isPodReady(pod)

		shouldDelete := false
		shouldCreate := false

		// Log raw event stream to debug log
		switch event.Type {
		case watch.Added:
			log.Printf("ADDED pod %s with ip %s. Ready = %v", podName, podIp, podReady)
			if podReady {
				shouldCreate = true
			} else {
				shouldDelete = true
			}

		case watch.Modified:
			log.Printf("MODIFIED pod %s with ip %s. Ready = %v", podName, podIp, podReady)
			if podReady {
				shouldCreate = true
			} else {
				shouldDelete = true
			}
		case watch.Deleted:
			log.Printf("DELETED pod %s with ip %s. Ready = %v", podName, podIp, podReady)
			shouldDelete = true
		default:
			// ignore this event
			continue
		}

		if shouldDelete {
			pool.RemoveRecipe(podName)
			delete(p.pods, podName)
		} else if shouldCreate {
			r := p.BaseRecipe
			r.Address = fmt.Sprintf(r.Address, pod.Status.PodIP)
			pool.AddRecipe(podName, r)
		}

	}

	return nil
}

func isPodReady(pod *v1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}
