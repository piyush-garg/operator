package helpers

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	op "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/test/tektonpipeline"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// AssertNoError confirms the error returned is nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// WaitForDeploymentDeletion checks to see if a given deployment is deleted
// the function returns an error if the given deployment is not deleted within the timeout
func WaitForDeploymentDeletion(t *testing.T, namespace, name string) error {
	t.Helper()

	err := wait.Poll(tektonpipeline.APIRetry, tektonpipeline.APITimeout, func() (bool, error) {
		kc := test.Global.KubeClient
		_, err := kc.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsGone(err) || apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}

		t.Logf("Waiting for deletion of %s deployment\n", name)
		return false, nil
	})
	if err == nil {
		t.Logf("%s Deployment deleted\n", name)
	}
	return err
}

func WaitForClusterCR(t *testing.T, name string, obj runtime.Object) {
	t.Helper()

	objKey := types.NamespacedName{Name: name}

	err := wait.Poll(tektonpipeline.APIRetry, tektonpipeline.APITimeout, func() (bool, error) {
		err := test.Global.Client.Get(context.TODO(), objKey, obj)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s cr\n", name)
				return false, nil
			}
			return false, err
		}

		return true, nil
	})

	AssertNoError(t, err)
}

func DeletePipelineDeployment(t *testing.T, dep *appsv1.Deployment) {
	t.Helper()
	err := wait.Poll(tektonpipeline.APIRetry, tektonpipeline.APITimeout, func() (bool, error) {
		err := test.Global.Client.Delete(context.TODO(), dep)
		if err != nil {
			t.Logf("Deletion of deployment %s failed %s \n", dep.GetName(), err)
			return false, err
		}

		return true, nil
	})

	AssertNoError(t, err)
}

func DeleteClusterCR(t *testing.T, name string) {
	t.Helper()

	// ensure object exists before deletion
	objKey := types.NamespacedName{Name: name}
	cr := &op.TektonPipeline{}
	err := test.Global.Client.Get(context.TODO(), objKey, cr)
	if err != nil {
		t.Logf("Failed to find cluster CR: %s : %s\n", name, err)
	}
	AssertNoError(t, err)

	err = wait.Poll(tektonpipeline.APIRetry, tektonpipeline.APITimeout, func() (bool, error) {
		err := test.Global.Client.Delete(context.TODO(), cr)
		if err != nil {
			t.Logf("Deletion of CR %s failed %s \n", name, err)
			return false, err
		}

		return true, nil
	})

	AssertNoError(t, err)
}

func ValidatePipelineSetup(t *testing.T, cr *op.TektonPipeline, deployments ...string) {
	t.Helper()

	kc := test.Global.KubeClient
	ns := cr.Spec.TargetNamespace

	for _, d := range deployments {
		err := e2eutil.WaitForDeployment(
			t, kc, ns,
			d,
			1,
			tektonpipeline.APIRetry,
			tektonpipeline.APITimeout,
		)
		AssertNoError(t, err)
	}
}

func ValidatePipelineCleanup(t *testing.T, cr *op.TektonPipeline, deployments ...string) {
	t.Helper()

	ns := cr.Spec.TargetNamespace
	for _, d := range deployments {
		err := WaitForDeploymentDeletion(t, ns, d)
		AssertNoError(t, err)
	}
}

func DeployOperator(t *testing.T, ctx *test.TestCtx) error {
	err := ctx.InitializeClusterResources(
		&test.CleanupOptions{
			TestContext:   ctx,
			Timeout:       tektonpipeline.CleanupTimeout,
			RetryInterval: tektonpipeline.CleanupRetry,
		},
	)
	if err != nil {
		return err
	}

	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}

	return e2eutil.WaitForOperatorDeployment(
		t,
		test.Global.KubeClient,
		namespace,
		tektonpipeline.TestOperatorName,
		1,
		tektonpipeline.APIRetry,
		tektonpipeline.APITimeout,
	)
}
