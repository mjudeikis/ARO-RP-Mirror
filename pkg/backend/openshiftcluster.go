package backend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/backend/openshiftcluster"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type openShiftClusterBackend struct {
	*backend
}

// try tries to dequeue an OpenShiftClusterDocument for work, and works it on a
// new goroutine.  It returns a boolean to the caller indicating whether it
// succeeded in dequeuing anything - if this is false, the caller should sleep
// before calling again
func (ocb *openShiftClusterBackend) try() (bool, error) {
	doc, err := ocb.db.OpenShiftClusters.Dequeue()
	if err != nil || doc == nil {
		return false, err
	}

	log := ocb.baseLog.WithField("resource", doc.OpenShiftCluster.ID)
	if doc.Dequeues > maxDequeueCount {
		log.Errorf("dequeued %d times, failing", doc.Dequeues)
		return true, ocb.endLease(nil, doc, api.ProvisioningStateFailed)
	}

	log.Print("dequeued")
	atomic.AddInt32(&ocb.workers, 1)
	go func() {
		defer recover.Panic(log)

		defer func() {
			atomic.AddInt32(&ocb.workers, -1)
			ocb.cond.Signal()
		}()

		t := time.Now()

		err := ocb.handle(context.Background(), log, doc)
		if err != nil {
			log.Error(err)
		}
		log.WithField("durationMs", int(time.Now().Sub(t)/time.Millisecond)).Print("done")
	}()

	return true, nil
}

// handle is responsible for handling backend operation and lease
func (ocb *openShiftClusterBackend) handle(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stop := ocb.heartbeat(cancel, log, doc)
	defer stop()

	m, err := openshiftcluster.NewManager(log, ocb.env, ocb.db.OpenShiftClusters, doc)
	if err != nil {
		log.Error(err)
		return ocb.endLease(stop, doc, api.ProvisioningStateFailed)
	}
	currentState := doc.OpenShiftCluster.Properties.ProvisioningState
	switch doc.OpenShiftCluster.Properties.ProvisioningState {
	case api.ProvisioningStateCreating:
		log.Print("creating")

		err = m.Create(ctx)
		if err != nil {
			log.Error(err)
			ocb.updateState(state)
			return ocb.endLease(stop, doc, api.ProvisioningStateFailed, expectedState)
		}

		return ocb.endLease(stop, doc, api.ProvisioningStateSucceeded)

	case api.ProvisioningStateUpdating:
		log.Print("updating")

		err = m.Update(ctx)
		if err != nil {
			log.Error(err)
			return ocb.endLease(stop, doc, api.ProvisioningStateFailed)
		}

		return ocb.endLease(stop, doc, api.ProvisioningStateSucceeded)

	case api.ProvisioningStateDeleting:
		log.Print("deleting")

		err = m.Delete(ctx)
		if err != nil {
			log.Error(err)
			return ocb.endLease(stop, doc, api.ProvisioningStateFailed)
		}

		err = ocb.updateAsyncOperation(doc.AsyncOperationID, nil, api.ProvisioningStateSucceeded, "")
		if err != nil {
			log.Error(err)
			return ocb.endLease(stop, doc, api.ProvisioningStateFailed)
		}

		stop()

		return ocb.db.OpenShiftClusters.Delete(doc)
	}

	return fmt.Errorf("unexpected provisioningState %q", doc.OpenShiftCluster.Properties.ProvisioningState)
}

func (ocb *openShiftClusterBackend) heartbeat(cancel context.CancelFunc, log *logrus.Entry, doc *api.OpenShiftClusterDocument) func() {
	var stopped bool
	stop, done := make(chan struct{}), make(chan struct{})

	go func() {
		defer recover.Panic(log)

		defer close(done)

		t := time.NewTicker(10 * time.Second)
		defer t.Stop()

		for {
			current, err := ocb.db.OpenShiftClusters.Get(doc.Key)
			if err != nil {
				log.Error(err)
				return
			}
			// If current document, we are working on is not in Deleting
			// but frontend updated to delete it - cancel and release
			if doc.OpenShiftCluster.Properties.ProvisioningState != api.ProvisioningStateDeleting &&
				current.OpenShiftCluster.Properties.ProvisioningState == api.ProvisioningStateDeleting {
				cancel()
				go ocb.handleDelete(current)
			}

			_, err = ocb.db.OpenShiftClusters.Lease(doc.Key)
			if err != nil {
				cancel()
				log.Error(err)
				return
			}

			select {
			case <-t.C:
			case <-stop:
				return
			}
		}
	}()

	return func() {
		if !stopped {
			close(stop)
			<-done
			stopped = true
		}
	}
}

func (ocb *openShiftClusterBackend) updateAsyncOperation(id string, oc *api.OpenShiftCluster, provisioningState, failedProvisioningState api.ProvisioningState) error {
	if id != "" {
		_, err := ocb.db.AsyncOperations.Patch(id, func(asyncdoc *api.AsyncOperationDocument) error {
			asyncdoc.AsyncOperation.ProvisioningState = provisioningState

			now := time.Now()
			asyncdoc.AsyncOperation.EndTime = &now

			if provisioningState == api.ProvisioningStateFailed {
				asyncdoc.AsyncOperation.Error = &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInternalServerError,
					Message: "Internal server error.",
				}
			}

			if oc != nil {
				ocCopy := *oc
				ocCopy.Properties.ProvisioningState = provisioningState
				ocCopy.Properties.FailedProvisioningState = failedProvisioningState

				asyncdoc.OpenShiftCluster = &ocCopy
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (ocb *openShiftClusterBackend) endLease(stop func(), doc *api.OpenShiftClusterDocument, provisioningState api.ProvisioningState, currentExpected api.ProvisioningState) error {
	//var failedProvisioningState api.ProvisioningState
	//if provisioningState == api.ProvisioningStateFailed {
	//	failedProvisioningState = doc.OpenShiftCluster.Properties.ProvisioningState
	//}

	err := ocb.updateAsyncOperation(doc.AsyncOperationID, doc.OpenShiftCluster, provisioningState, failedProvisioningState)
	if err != nil {
		return err
	}

	if stop != nil {
		stop()
	}

	// get from the document in the patch in db layer
	_, err = ocb.db.OpenShiftClusters.EndLease(doc.Key, provisioningState, currentExpected)
	return err
}

// handleDelete handles document state during context cancelation and backend
// worker shutdown. If backend worker context is canceled, it will mark document
// as Failed. We need to wait for it and update to Deleting.
// End to end state change flow should be:
// Creating -> Deleting -> Failed -> Deleting
func (ocb *openShiftClusterBackend) handleDelete(doc *api.OpenShiftClusterDocument) {
	ocb.baseLog.Print("handle delete")
	err := ocb.updateAsyncOperation(doc.AsyncOperationID, nil, api.ProvisioningStateDeleting, "")
	if err != nil {
		ocb.baseLog.Error(err)
	}

	// we can skip failed state if backends cancels in order
	err = wait.Poll(time.Millisecond*500, time.Minute, func() (bool, error) {
		current, err := ocb.db.OpenShiftClusters.Get(doc.Key)
		if err != nil {
			return false, err
		}
		spew.Dump(current.OpenShiftCluster.Properties.ProvisioningState)
		if current.OpenShiftCluster.Properties.ProvisioningState == api.ProvisioningStateFailed {
			err = ocb.updateAsyncOperation(doc.AsyncOperationID, doc.OpenShiftCluster, api.ProvisioningStateDeleting, "")
			if err != nil {
				ocb.baseLog.Error(err)
			}
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		ocb.baseLog.Error(err)
	}
}
