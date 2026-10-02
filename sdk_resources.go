// Hand-written companion to sdk.go. The codegen-emitted sdk.go covers
// generated services + models + Layer-2 wrappers; this file re-exports
// hand-coded helpers under resources/ so consumers can reach the entire
// public SDK surface via a single `sdk` import. clean-generated leaves
// this file alone (it removes only sdk.go).

package sdk

import (
	"github.com/euc-oss/terraform-sdk-uem/resources"
)

// BaselineCreator is the multipart-create helper for the Baselines V1
// API. See resources/baseline_creator.go for the workflow seam this
// helper bridges and its retirement criterion.
type BaselineCreator = resources.BaselineCreator

// NewBaselineCreator constructs a BaselineCreator backed by the given
// SDK client. Use this for POST /api/mdm/groups/{ogUuid}/baselines
// only; every other Baselines operation lives on the generated
// BaselinesV1Service.
var NewBaselineCreator = resources.NewBaselineCreator
