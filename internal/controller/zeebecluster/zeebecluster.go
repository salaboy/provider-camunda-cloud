/*
Copyright 2020 The Crossplane Authors.

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

package zeebecluster

import (
	"context"
	"fmt"
	cc "github.com/camunda-community-hub/camunda-cloud-go-client/pkg/cc/client"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	sdktraceresource "go.opentelemetry.io/otel/sdk/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/salaboy/provider-camunda-cloud/apis/cc/v1alpha1"
	apisv1alpha1 "github.com/salaboy/provider-camunda-cloud/apis/v1alpha1"
)

const (
	errNotMyType    = "managed resource is not a MyType custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errCannotLoginToCC = "cannot login to Camunda Cloud"
	errNewClient = "cannot create new Service"
)

type CCCredentials struct{
	CCClientId string `json:"ccClientId"`
	CCSecretId string `json:"ccSecretId"`
}


// Setup adds a controller that reconciles MyType managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.ZeebeClusterGroupKind)

	o := controller.Options{
		RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ZeebeClusterGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.ZeebeCluster{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        resource.Tracker
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {

	cr, ok := mg.(*v1alpha1.ZeebeCluster)
	if !ok {
		return nil, errors.New(errNotMyType)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	credentials := 	CCCredentials{}

	err = json.Unmarshal(data, &credentials)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	var svc = cc.CCClient{}

	flush := svc.InitTracer()
	defer flush()
	var logged, _ = svc.LoginWithContext(ctx, credentials.CCClientId, credentials.CCSecretId)

	flush2 := initTracer()
	defer flush2()

	if logged{
		fmt.Printf("logged in!\n")
	}else{
		return nil, errors.New(errCannotLoginToCC)
	}

	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}


	return &external{service: svc, tracer: otel.Tracer("provider-camunda-cloud")}, nil
}

func initTracer() func() {

	// Create and install Jaeger export pipeline.
	flush, err := jaeger.InstallNewPipeline(
		jaeger.WithCollectorEndpoint("http://localhost:14268/api/traces"),
		jaeger.WithSDKOptions(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithResource(sdktraceresource.NewWithAttributes(
				semconv.ServiceNameKey.String("provider-camunda-cloud"),
				attribute.String("exporter", "jaeger"),
				attribute.Float64("float", 312.23),
			)),
		),
	)
	if err != nil {
		fmt.Errorf("Failed to initialize Tracer",err)
	}
	return flush
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	service cc.CCClient
	tracer  trace.Tracer
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {

	ctx, span := e.tracer.Start(ctx, "observe")
	defer span.End()

	cr, ok := mg.(*v1alpha1.ZeebeCluster)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotMyType)
	}

	// These fmt statements should be removed in the real implementation.
	fmt.Printf("Observing: %+v\n", cr)


	fmt.Printf("Querying for Cluster Name: %s\n", cr.Name)
	existing, err := e.service.GetClusterByNameWithContext(ctx, cr.Name)

	fmt.Printf("Existing Cluster ID: %s\n", existing.ID)
	//if database.IsNotFound(err) {
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, err
	}
	if existing.ID == "" && cr.Status.AtProvider.ClusterId == "" {
		return managed.ExternalObservation{ResourceExists: false, ResourceUpToDate: true, ConnectionDetails: managed.ConnectionDetails{}}, nil
	} else {

		if cr.Spec.ForProvider.PlanName != existing.ClusterPlantType.Name{
			cr.Spec.ForProvider.PlanName = existing.ClusterPlantType.Name
		}

		if cr.Spec.ForProvider.GenerationName != existing.Generation.Name{
			cr.Spec.ForProvider.GenerationName = existing.Generation.Name
		}

		if cr.Spec.ForProvider.ChannelName != existing.Channel.Name {
			cr.Spec.ForProvider.ChannelName = existing.Channel.Name
		}
		if cr.Spec.ForProvider.Region != existing.K8sContext.Name{
			cr.Spec.ForProvider.Region = existing.K8sContext.Name
		}

		cr.Status.AtProvider.ClusterId = existing.ID
		clusterStatus, err := e.service.GetClusterDetailsWithContext(ctx, existing.ID)
		if err != nil {
			cr.SetConditions(xpv1.Unavailable())
			return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false, ConnectionDetails: managed.ConnectionDetails{}}, nil
		}
		cr.Status.AtProvider.ClusterStatus = clusterStatus
		fmt.Printf("CLUSTER STATUS: %s\n", cr.Status.AtProvider.ClusterStatus.Ready)
		switch cr.Status.AtProvider.ClusterStatus.Ready {
		case "Healthy":
			cr.SetConditions(xpv1.Available())
		case "Creating":
			cr.SetConditions(xpv1.Creating())
		case "Not Healthy":
			cr.SetConditions(xpv1.Unavailable())
		}
		return managed.ExternalObservation{
			// Return false when the external resource does not exist. This lets
			// the managed resource reconciler know that it needs to call Create to
			// (re)create the resource, or that it has successfully been deleted.
			ResourceExists: true,

			// Return false when the external resource exists, but it not up to date
			// with the desired managed resource state. This lets the managed
			// resource reconciler know that it needs to call Update.
			ResourceUpToDate: true,

			// Return any details that may be required to connect to the external
			// resource. These will be stored as the connection secret.
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,

		// Return any details that may be required to connect to the external
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {

	ctx, span := e.tracer.Start(ctx, "create")
	defer span.End()

	cr, ok := mg.(*v1alpha1.ZeebeCluster)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotMyType)
	}

	fmt.Printf("Creating: %+v\n", cr)

	e.service.GetClusterParamsWithContext(ctx)

	clusterId, err := e.service.CreateClusterWithParamsAndContext(ctx, mg.GetName(), cr.Spec.ForProvider.PlanName,
		cr.Spec.ForProvider.ChannelName, cr.Spec.ForProvider.GenerationName, cr.Spec.ForProvider.Region)
	if err != nil {
		fmt.Errorf("failed to create zeebe cluster %s\n", err.Error())
		return managed.ExternalCreation{}, err
	}
	fmt.Printf("Updating Zeebe Cluster with ClusterId: %s\n", clusterId)

	cr.Status.AtProvider.ClusterId = clusterId

	cr.SetConditions(xpv1.Creating())

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	ctx, span := e.tracer.Start(ctx, "update")
	defer span.End()

	cr, ok := mg.(*v1alpha1.ZeebeCluster)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotMyType)
	}

	fmt.Printf("Updating: %+v", cr)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	ctx, span := e.tracer.Start(ctx, "delete")
	defer span.End()

	cr, ok := mg.(*v1alpha1.ZeebeCluster)
	if !ok {
		return errors.New(errNotMyType)
	}

	fmt.Printf("Deleting: %+v", cr)

	deleted, err := e.service.DeleteClusterWithContext(ctx, cr.Status.AtProvider.ClusterId)
	if err != nil {
		fmt.Printf("Failed to delete cluster: cluster not found %s", err)
	}
	if deleted {
		fmt.Printf("Cluster in camunda cloud deleted: %s ", cr.Status.AtProvider.ClusterId)
	}

	return nil
}
