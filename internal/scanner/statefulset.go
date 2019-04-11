package scanner

import (
	"fmt"

	"github.com/golang/glog"
	v1beta "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	appsv1beta "k8s.io/client-go/kubernetes/typed/apps/v1beta1"
	"k8s.io/client-go/rest"
)

type StatefulSetScanner struct {
	config     Config
	kubernetes *rest.Config
}

func init() {
	RegisterModule("statefulset", NewStatefulSetScanner)
}

// NewStatefulSetScanner will instantiate a new StatefulSetScanner object.
func NewStatefulSetScanner() Scanner {
	kubernetes, err := getKubernetes()
	if err != nil {
		glog.Warningf("failed instantiating k8s client: %s", err)
	}
	return &StatefulSetScanner{
		kubernetes: kubernetes,
	}
}

// SetConfig will set the generic configuration for this scanner.
func (s *StatefulSetScanner) SetConfig(cfg Config) {
	s.config = cfg
}

// SetConfig will set the generic configuration for this scanner.
func (s *StatefulSetScanner) GetConfig() Config {
	return s.config
}

// GetObjects will return a populated list of Objects containing the relavant
// resources with their schedule info.
func (s *StatefulSetScanner) GetObjects() ([]*Object, error) {
	rcs, err := s.getStatefulSets()
	if err != nil {
		return nil, err
	}
	return s.getObjects(rcs)
}

// Scale will scale a given object to given amount of replicas.
func (s *StatefulSetScanner) Scale(obj *Object, replicas int) error {
	glog.Infof("Scaling %s/%s to %d replicas", obj.Namespace, obj.Name, replicas)
	ss, err := s.getStatefulSet(obj)
	if err != nil {
		return fmt.Errorf("GetScale failed with: %s", err)
	}
	repl := int32(replicas)
	ss.Spec.Replicas = &repl
	apps, _ := appsv1beta.NewForConfig(s.kubernetes)
	_, err = apps.StatefulSets(obj.Namespace).Update(ss)
	return nil
}

// SaveState will save the current number of replicas as an annotation on the
// statefulset config.
func (s *StatefulSetScanner) SaveState(obj *Object) (int, error) {
	ss, err := s.getStatefulSet(obj)
	if err != nil {
		return 0, err
	}
	repl := int(*ss.Spec.Replicas)
	ss.ObjectMeta = updateState(ss.ObjectMeta, repl)
	apps, _ := appsv1beta.NewForConfig(s.kubernetes)
	_, err = apps.StatefulSets(obj.Namespace).Update(ss)
	return repl, err
}

// getStatefulSet will return the statefulset for given object.
func (s *StatefulSetScanner) getStatefulSet(obj *Object) (*v1beta.StatefulSet, error) {
	if s.kubernetes == nil {
		return nil, fmt.Errorf("unable to connect to kubernetes")
	}
	apps, err := appsv1beta.NewForConfig(s.kubernetes)
	if err != nil {
		return nil, err
	}
	return apps.StatefulSets(obj.Namespace).Get(obj.Name, metav1.GetOptions{})
}

// getStatefulSets will return all statefulsets in the namespace that
// match the label selector.
func (s *StatefulSetScanner) getStatefulSets() (*v1beta.StatefulSetList, error) {
	if s.kubernetes == nil {
		return nil, fmt.Errorf("unable to connect to kubernetes")
	}
	apps, err := appsv1beta.NewForConfig(s.kubernetes)
	if err != nil {
		return nil, err
	}
	return apps.StatefulSets(s.config.Namespace).List(metav1.ListOptions{
		LabelSelector: s.config.Label,
	})
}

// getObjects will itterate through the list of deployment configs and populate
// a list of objects containing the schedule configuration (if any).
func (s *StatefulSetScanner) getObjects(rcs *v1beta.StatefulSetList) ([]*Object, error) {
	objs := []*Object{}
	for _, rc := range rcs.Items {
		if obj := s.getObject(&rc); obj.Schedule != nil {
			objs = append(objs, obj)
		}
	}
	return objs, nil
}

// Watch will return a channel on which Event objects will be published that
// describe change events in the cluster.
func (s *StatefulSetScanner) Watch(_stop chan bool) (chan Event, error) {
	if s.kubernetes == nil {
		return nil, fmt.Errorf("unable to connect to kubernetes")
	}
	apps, err := appsv1beta.NewForConfig(s.kubernetes)
	if err != nil {
		return nil, err
	}
	watcher, err := apps.StatefulSets(s.config.Namespace).Watch(metav1.ListOptions{
		LabelSelector: s.config.Label,
	})
	if err != nil {
		return nil, err
	}

	out := make(chan Event)
	go func() {
		inch := watcher.ResultChan()
		for {
			select {
			case evt := <-inch:
				glog.V(5).Infof("Received event: %v", evt)
				s.handleEvent(out, evt)
			case <-_stop:
				return
			}
		}
	}()

	return out, nil
}

// handleEvent will take a watch event and transform it to a scanner watch
// event, and publish it to the out channel.
func (s *StatefulSetScanner) handleEvent(out chan Event, evt watch.Event) {
	if evt.Type == watch.Error {
		glog.Errorf("Error watching: %v", evt)
		return
	}

	ss, ok := evt.Object.(*v1beta.StatefulSet)
	if !ok {
		glog.Errorf("Unexpected type; %v", ss)
		return
	}

	obj := s.getObject(ss)
	if evt.Type == watch.Deleted {
		out <- Event{Object: obj, Type: EventRemove}
		return
	}

	if evt.Type == watch.Added || evt.Type == watch.Modified {
		if obj.Schedule != nil {
			out <- Event{Object: obj, Type: EventAdd}
		} else {
			out <- Event{Object: obj, Type: EventRemove}
		}
	}
}

// getObject will convert a deploymentconfig object to a scanner.Object.
func (s *StatefulSetScanner) getObject(rc *v1beta.StatefulSet) *Object {
	obj := NewObjectForScanner(s)
	if err := obj.updateForMeta(rc.ObjectMeta); err != nil {
		glog.Error(err)
	}
	obj.Replicas = int(*rc.Spec.Replicas)
	return obj
}
