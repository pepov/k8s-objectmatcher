module github.com/banzaicloud/k8s-objectmatcher

go 1.12

require (
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/goph/emperror v0.17.2
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/pkg/errors v0.8.1
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4 // indirect
	golang.org/x/net v0.0.0-20190724013045-ca1201d0de80 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 // indirect
	golang.org/x/sys v0.0.0-20190804053845-51ab0e2deafa // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/appengine v1.6.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/api v0.0.0-20190809220925-3ab596449d6f
	k8s.io/apiextensions-apiserver v0.0.0-20190810101755-ebc439d6a67b
	k8s.io/apimachinery v0.0.0-20190809020650-423f5d784010
	k8s.io/client-go v11.0.1-0.20190516230509-ae8359b20417+incompatible
	k8s.io/klog v0.4.0
	k8s.io/kube-openapi v0.0.0-20190722073852-5e22f3d471e6 // indirect
	k8s.io/utils v0.0.0-20190809000727-6c36bc71fc4a // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)

// pin k8s release 1.14

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190704095032-f4ca3d3bdf1d
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190801143813-8b5f3a974f92
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190704094733-8f6ac2502e51
)
