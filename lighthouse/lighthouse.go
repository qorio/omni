package lighthouse

import (
	"github.com/golang/glog"
)

type serviceImpl struct {
	settings Settings
}

func NewService(settings Settings) (Service, error) {

	impl := &serviceImpl{
		settings: settings,
	}
	return impl, nil
}

func (this *serviceImpl) Close() {
	glog.Infoln("Service closed")
}
