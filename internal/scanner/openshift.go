package scanner

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	v1 "github.com/openshift/api/apps/v1"
	appsv1 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/joyrex2001/nightshift/internal/schedule"
)

type OpenShiftScanner struct {
	config     Config
	kubernetes *rest.Config
}

// NewOpenShiftScanner will instantiate a new OpenShiftScanner object.
func NewOpenShiftScanner() *OpenShiftScanner {
	kubernetes, err := getKubernetes()
	if err != nil {
		glog.Warningf("failed instantiating k8s client: %s", err)
	}
	return &OpenShiftScanner{
		kubernetes: kubernetes,
	}
}

// getKubernetes will return a kubernetes config object.
func getKubernetes() (*rest.Config, error) {
	kubeconfig := viper.GetString("openshift.kubeconfig")
	if kubeconfig != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err == nil {
			return config, nil
		}
	}
	return rest.InClusterConfig()
}

// SetConfig will set the generic configuration for this scanner.
func (s *OpenShiftScanner) SetConfig(cfg Config) {
	cfg.Type = OpenShift
	s.config = cfg
}

// SetConfig will set the generic configuration for this scanner.
func (s *OpenShiftScanner) GetConfig() Config {
	return s.config
}

// GetObjects will return a populated list of Objects containing the relavant
// resources with their schedule info.
func (s *OpenShiftScanner) GetObjects() ([]*Object, error) {
	rcs, err := s.getDeploymentConfigs()
	if err != nil {
		return nil, err
	}
	return s.getObjects(rcs)
}

// Scale will scale a given object to given amount of replicas.
func (s *OpenShiftScanner) Scale(obj *Object, replicas int) error {
	glog.Infof("Scaling %s/%s to %d replicas", obj.Namespace, obj.Name, replicas)
	if s.kubernetes == nil {
		return fmt.Errorf("unable to connect to kubernetes")
	}
	apps, err := appsv1.NewForConfig(s.kubernetes)
	if err != nil {
		return err
	}
	scale, err := apps.DeploymentConfigs(obj.Namespace).GetScale(obj.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("GetScale failed with: %s", err)
	}
	scale.Spec.Replicas = int32(replicas)
	_, err = apps.DeploymentConfigs(obj.Namespace).UpdateScale(obj.Name, scale)
	return err
}

// LoadState will load the State in an object with a the value of the State
// annoation on the deployment config.
func (s *OpenShiftScanner) LoadState(obj *Object) error {
	glog.Infof("TODO: LoadState")
	return nil
}

// SaveState will save the current number of replicas as an annotation on the
// deployment config.
func (s *OpenShiftScanner) SaveState(obj *Object) error {
	glog.Infof("TODO: SaveState")
	return nil
}

// getDeploymentConfigs will return all deploymentconfigs in the namespace that
// match the label selector.
func (s *OpenShiftScanner) getDeploymentConfigs() (*v1.DeploymentConfigList, error) {
	if s.kubernetes == nil {
		return nil, fmt.Errorf("unable to connect to kubernetes")
	}
	apps, err := appsv1.NewForConfig(s.kubernetes)
	if err != nil {
		return nil, err
	}
	return apps.DeploymentConfigs(s.config.Namespace).List(metav1.ListOptions{
		LabelSelector: s.config.Label,
	})
}

// getObjects will itterate through the list of deployment configs and populate
// a list of objects containing the schedule configuration (if any).
func (s *OpenShiftScanner) getObjects(rcs *v1.DeploymentConfigList) ([]*Object, error) {
	objs := []*Object{}
	for _, rc := range rcs.Items {
		sched, err := s.getSchedule(rc.ObjectMeta.Annotations)
		if err != nil {
			glog.Errorf("error parsing schedule annotation for %s (%s); %s", rc.ObjectMeta.UID, rc.ObjectMeta.Name, err)
		}
		if sched != nil {
			objs = append(objs, &Object{
				Name:      rc.ObjectMeta.Name,
				Namespace: s.config.Namespace,
				UID:       string(rc.ObjectMeta.UID),
				Type:      OpenShift,
				Schedule:  sched,
				Replicas:  int(rc.Spec.Replicas),
			})
		}
	}
	return objs, nil
}

// getSchedule will return a list of schedules, taken the annotations and
// defaults into account.
func (s *OpenShiftScanner) getSchedule(annotations map[string]string) ([]*schedule.Schedule, error) {
	dis := strings.ToLower(annotations["joyrex2001.com/nightshift.ignore"])
	if dis == "true" {
		return nil, nil
	} else if dis != "false" && dis != "" {
		return nil, fmt.Errorf("invalid value '%s' for nightshift.ignore", dis)
	}
	if ann := annotations["joyrex2001.com/nightshift.schedule"]; ann != "" {
		return s.annotationToSchedule(ann)
	}
	return s.config.Schedule, nil
}

// annotationToSchedule will convert the contents of the schedule annotation
// to an array of Schedule objects. It will produce an error if the provided
// annotation value is invalid.
func (s *OpenShiftScanner) annotationToSchedule(annotation string) ([]*schedule.Schedule, error) {
	sched := []*schedule.Schedule{}
	for _, ann := range strings.Split(annotation, ";") {
		if ann == "" {
			continue
		}
		s, err := schedule.New(ann)
		if err != nil {
			return nil, err
		}
		sched = append(sched, s)
	}
	return sched, nil
}
