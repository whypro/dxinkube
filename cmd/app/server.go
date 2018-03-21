package app

import (
	"github.com/golang/glog"

	"github.com/whypro/dxinkube/pkg/controller"
	"k8s.io/apiserver/pkg/server"
)

func Run(zkControllerOptions *ZKControllerOptions) (err error) {

	// router := gin.Default()

	zkControllerConfig := createZKControllerConfig(zkControllerOptions)
	zkController, err := controller.NewZKController(zkControllerConfig)
	if err != nil {
		glog.Errorf("create zk controller error, err: %v", err)
		return err
	}

	stopCh := server.SetupSignalHandler()

	zkController.Run(stopCh)

	<-stopCh
	glog.Infof("shutting down http server")

	glog.Infof("alert-dispatcher shutdown success")

	return nil
}
