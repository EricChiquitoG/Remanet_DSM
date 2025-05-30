// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.2
// 	protoc        v5.29.0
// source: DSM.proto

package DSM_protos

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Process struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StepName         string                 `protobuf:"bytes,1,opt,name=step_name,json=stepName,proto3" json:"step_name,omitempty"` // Name of the step
	ProductType      string                 `protobuf:"bytes,2,opt,name=product_type,json=productType,proto3" json:"product_type,omitempty"`
	EconomicOperator string                 `protobuf:"bytes,3,opt,name=economic_operator,json=economicOperator,proto3" json:"economic_operator,omitempty"` // Name or identifier of the economic operator
	SubmittedAt      *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=submitted_at,json=submittedAt,proto3" json:"submitted_at,omitempty"`                // Submission timestamp (ISO 8601 format recommended)
	Requirements     []string               `protobuf:"bytes,5,rep,name=requirements,proto3" json:"requirements,omitempty"`                                 // List of requirements as strings
}

func (x *Process) Reset() {
	*x = Process{}
	mi := &file_DSM_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Process) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Process) ProtoMessage() {}

func (x *Process) ProtoReflect() protoreflect.Message {
	mi := &file_DSM_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Process.ProtoReflect.Descriptor instead.
func (*Process) Descriptor() ([]byte, []int) {
	return file_DSM_proto_rawDescGZIP(), []int{0}
}

func (x *Process) GetStepName() string {
	if x != nil {
		return x.StepName
	}
	return ""
}

func (x *Process) GetProductType() string {
	if x != nil {
		return x.ProductType
	}
	return ""
}

func (x *Process) GetEconomicOperator() string {
	if x != nil {
		return x.EconomicOperator
	}
	return ""
}

func (x *Process) GetSubmittedAt() *timestamppb.Timestamp {
	if x != nil {
		return x.SubmittedAt
	}
	return nil
}

func (x *Process) GetRequirements() []string {
	if x != nil {
		return x.Requirements
	}
	return nil
}

type ProcessResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Status     string   `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`   // Status of the submission (e.g., "success" or "failure")
	Message    string   `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"` // Additional details or error message
	Capability []string `protobuf:"bytes,3,rep,name=capability,proto3" json:"capability,omitempty"`
}

func (x *ProcessResponse) Reset() {
	*x = ProcessResponse{}
	mi := &file_DSM_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ProcessResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProcessResponse) ProtoMessage() {}

func (x *ProcessResponse) ProtoReflect() protoreflect.Message {
	mi := &file_DSM_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProcessResponse.ProtoReflect.Descriptor instead.
func (*ProcessResponse) Descriptor() ([]byte, []int) {
	return file_DSM_proto_rawDescGZIP(), []int{1}
}

func (x *ProcessResponse) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

func (x *ProcessResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *ProcessResponse) GetCapability() []string {
	if x != nil {
		return x.Capability
	}
	return nil
}

type Purchase struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ProductType  string    `protobuf:"bytes,1,opt,name=product_type,json=productType,proto3" json:"product_type,omitempty"`
	Amount       string    `protobuf:"bytes,2,opt,name=amount,proto3" json:"amount,omitempty"`
	Location     []float32 `protobuf:"fixed32,3,rep,packed,name=location,proto3" json:"location,omitempty"`
	Logistics    float64   `protobuf:"fixed64,4,opt,name=logistics,proto3" json:"logistics,omitempty"`
	Co2          float64   `protobuf:"fixed64,5,opt,name=co2,proto3" json:"co2,omitempty"`
	Energy       float64   `protobuf:"fixed64,6,opt,name=energy,proto3" json:"energy,omitempty"`
	CostEUR      float64   `protobuf:"fixed64,7,opt,name=costEUR,proto3" json:"costEUR,omitempty"`
	Requirements []string  `protobuf:"bytes,8,rep,name=requirements,proto3" json:"requirements,omitempty"`
}

func (x *Purchase) Reset() {
	*x = Purchase{}
	mi := &file_DSM_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Purchase) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Purchase) ProtoMessage() {}

func (x *Purchase) ProtoReflect() protoreflect.Message {
	mi := &file_DSM_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Purchase.ProtoReflect.Descriptor instead.
func (*Purchase) Descriptor() ([]byte, []int) {
	return file_DSM_proto_rawDescGZIP(), []int{2}
}

func (x *Purchase) GetProductType() string {
	if x != nil {
		return x.ProductType
	}
	return ""
}

func (x *Purchase) GetAmount() string {
	if x != nil {
		return x.Amount
	}
	return ""
}

func (x *Purchase) GetLocation() []float32 {
	if x != nil {
		return x.Location
	}
	return nil
}

func (x *Purchase) GetLogistics() float64 {
	if x != nil {
		return x.Logistics
	}
	return 0
}

func (x *Purchase) GetCo2() float64 {
	if x != nil {
		return x.Co2
	}
	return 0
}

func (x *Purchase) GetEnergy() float64 {
	if x != nil {
		return x.Energy
	}
	return 0
}

func (x *Purchase) GetCostEUR() float64 {
	if x != nil {
		return x.CostEUR
	}
	return 0
}

func (x *Purchase) GetRequirements() []string {
	if x != nil {
		return x.Requirements
	}
	return nil
}

type PurchaseResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Status     string `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
	Message    string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
	Capability bool   `protobuf:"varint,3,opt,name=capability,proto3" json:"capability,omitempty"`
}

func (x *PurchaseResponse) Reset() {
	*x = PurchaseResponse{}
	mi := &file_DSM_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PurchaseResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PurchaseResponse) ProtoMessage() {}

func (x *PurchaseResponse) ProtoReflect() protoreflect.Message {
	mi := &file_DSM_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PurchaseResponse.ProtoReflect.Descriptor instead.
func (*PurchaseResponse) Descriptor() ([]byte, []int) {
	return file_DSM_proto_rawDescGZIP(), []int{3}
}

func (x *PurchaseResponse) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

func (x *PurchaseResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *PurchaseResponse) GetCapability() bool {
	if x != nil {
		return x.Capability
	}
	return false
}

type Enroll struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name      string    `protobuf:"bytes,1,opt,name=Name,proto3" json:"Name,omitempty"`
	Address   string    `protobuf:"bytes,2,opt,name=Address,proto3" json:"Address,omitempty"`
	Location  []float64 `protobuf:"fixed64,3,rep,packed,name=Location,proto3" json:"Location,omitempty"`
	CostH     float64   `protobuf:"fixed64,4,opt,name=CostH,proto3" json:"CostH,omitempty"`
	Offerings []string  `protobuf:"bytes,5,rep,name=Offerings,proto3" json:"Offerings,omitempty"`
}

func (x *Enroll) Reset() {
	*x = Enroll{}
	mi := &file_DSM_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Enroll) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Enroll) ProtoMessage() {}

func (x *Enroll) ProtoReflect() protoreflect.Message {
	mi := &file_DSM_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Enroll.ProtoReflect.Descriptor instead.
func (*Enroll) Descriptor() ([]byte, []int) {
	return file_DSM_proto_rawDescGZIP(), []int{4}
}

func (x *Enroll) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Enroll) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *Enroll) GetLocation() []float64 {
	if x != nil {
		return x.Location
	}
	return nil
}

func (x *Enroll) GetCostH() float64 {
	if x != nil {
		return x.CostH
	}
	return 0
}

func (x *Enroll) GetOfferings() []string {
	if x != nil {
		return x.Offerings
	}
	return nil
}

type EnrollResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Status  string `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *EnrollResponse) Reset() {
	*x = EnrollResponse{}
	mi := &file_DSM_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EnrollResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnrollResponse) ProtoMessage() {}

func (x *EnrollResponse) ProtoReflect() protoreflect.Message {
	mi := &file_DSM_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnrollResponse.ProtoReflect.Descriptor instead.
func (*EnrollResponse) Descriptor() ([]byte, []int) {
	return file_DSM_proto_rawDescGZIP(), []int{5}
}

func (x *EnrollResponse) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

func (x *EnrollResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type OptimizationRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Structure should match the data needed by your Python optimization code
	// Based on your previous discussion, this might include:
	// Or you could structure it more granularly:
	JsonProblemData string `protobuf:"bytes,1,opt,name=json_problem_data,json=jsonProblemData,proto3" json:"json_problem_data,omitempty"` // Send the entire problem JSON as a string
}

func (x *OptimizationRequest) Reset() {
	*x = OptimizationRequest{}
	mi := &file_DSM_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OptimizationRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OptimizationRequest) ProtoMessage() {}

func (x *OptimizationRequest) ProtoReflect() protoreflect.Message {
	mi := &file_DSM_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OptimizationRequest.ProtoReflect.Descriptor instead.
func (*OptimizationRequest) Descriptor() ([]byte, []int) {
	return file_DSM_proto_rawDescGZIP(), []int{6}
}

func (x *OptimizationRequest) GetJsonProblemData() string {
	if x != nil {
		return x.JsonProblemData
	}
	return ""
}

// Message for the response (data sent from Python back to Go)
type OptimizationResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Structure should match the results you want to send back
	// This could be the Pareto front solutions and objectives
	Solutions    []*Solution `protobuf:"bytes,1,rep,name=solutions,proto3" json:"solutions,omitempty"`
	ErrorMessage string      `protobuf:"bytes,2,opt,name=error_message,json=errorMessage,proto3" json:"error_message,omitempty"` // Optional field for error handling
}

func (x *OptimizationResponse) Reset() {
	*x = OptimizationResponse{}
	mi := &file_DSM_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OptimizationResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OptimizationResponse) ProtoMessage() {}

func (x *OptimizationResponse) ProtoReflect() protoreflect.Message {
	mi := &file_DSM_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OptimizationResponse.ProtoReflect.Descriptor instead.
func (*OptimizationResponse) Descriptor() ([]byte, []int) {
	return file_DSM_proto_rawDescGZIP(), []int{7}
}

func (x *OptimizationResponse) GetSolutions() []*Solution {
	if x != nil {
		return x.Solutions
	}
	return nil
}

func (x *OptimizationResponse) GetErrorMessage() string {
	if x != nil {
		return x.ErrorMessage
	}
	return ""
}

// Message to represent a single solution from the optimization
type Solution struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UserIndices       []int32  `protobuf:"varint,1,rep,packed,name=user_indices,json=userIndices,proto3" json:"user_indices,omitempty"` // The sequence of user indices
	UserIds           []string `protobuf:"bytes,2,rep,name=user_ids,json=userIds,proto3" json:"user_ids,omitempty"`                     // The sequence of user IDs
	TransportCost     float64  `protobuf:"fixed64,3,opt,name=transport_cost,json=transportCost,proto3" json:"transport_cost,omitempty"`
	ManufacturingCost float64  `protobuf:"fixed64,4,opt,name=manufacturing_cost,json=manufacturingCost,proto3" json:"manufacturing_cost,omitempty"` // Add other relevant solution details if needed
}

func (x *Solution) Reset() {
	*x = Solution{}
	mi := &file_DSM_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Solution) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Solution) ProtoMessage() {}

func (x *Solution) ProtoReflect() protoreflect.Message {
	mi := &file_DSM_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Solution.ProtoReflect.Descriptor instead.
func (*Solution) Descriptor() ([]byte, []int) {
	return file_DSM_proto_rawDescGZIP(), []int{8}
}

func (x *Solution) GetUserIndices() []int32 {
	if x != nil {
		return x.UserIndices
	}
	return nil
}

func (x *Solution) GetUserIds() []string {
	if x != nil {
		return x.UserIds
	}
	return nil
}

func (x *Solution) GetTransportCost() float64 {
	if x != nil {
		return x.TransportCost
	}
	return 0
}

func (x *Solution) GetManufacturingCost() float64 {
	if x != nil {
		return x.ManufacturingCost
	}
	return 0
}

var File_DSM_proto protoreflect.FileDescriptor

var file_DSM_proto_rawDesc = []byte{
	0x0a, 0x09, 0x44, 0x53, 0x4d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x03, 0x44, 0x53, 0x4d,
	0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0xd9, 0x01, 0x0a, 0x07, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x12, 0x1b, 0x0a,
	0x09, 0x73, 0x74, 0x65, 0x70, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x08, 0x73, 0x74, 0x65, 0x70, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x21, 0x0a, 0x0c, 0x70, 0x72,
	0x6f, 0x64, 0x75, 0x63, 0x74, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0b, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x74, 0x54, 0x79, 0x70, 0x65, 0x12, 0x2b, 0x0a,
	0x11, 0x65, 0x63, 0x6f, 0x6e, 0x6f, 0x6d, 0x69, 0x63, 0x5f, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74,
	0x6f, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x10, 0x65, 0x63, 0x6f, 0x6e, 0x6f, 0x6d,
	0x69, 0x63, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x6f, 0x72, 0x12, 0x3d, 0x0a, 0x0c, 0x73, 0x75,
	0x62, 0x6d, 0x69, 0x74, 0x74, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0b, 0x73, 0x75,
	0x62, 0x6d, 0x69, 0x74, 0x74, 0x65, 0x64, 0x41, 0x74, 0x12, 0x22, 0x0a, 0x0c, 0x72, 0x65, 0x71,
	0x75, 0x69, 0x72, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x09, 0x52,
	0x0c, 0x72, 0x65, 0x71, 0x75, 0x69, 0x72, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x22, 0x63, 0x0a,
	0x0f, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x16, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x12, 0x1e, 0x0a, 0x0a, 0x63, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x79,
	0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0a, 0x63, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69,
	0x74, 0x79, 0x22, 0xe7, 0x01, 0x0a, 0x08, 0x50, 0x75, 0x72, 0x63, 0x68, 0x61, 0x73, 0x65, 0x12,
	0x21, 0x0a, 0x0c, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x74, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x74, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x6c, 0x6f,
	0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x03, 0x28, 0x02, 0x52, 0x08, 0x6c, 0x6f,
	0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1c, 0x0a, 0x09, 0x6c, 0x6f, 0x67, 0x69, 0x73, 0x74,
	0x69, 0x63, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x01, 0x52, 0x09, 0x6c, 0x6f, 0x67, 0x69, 0x73,
	0x74, 0x69, 0x63, 0x73, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x6f, 0x32, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x01, 0x52, 0x03, 0x63, 0x6f, 0x32, 0x12, 0x16, 0x0a, 0x06, 0x65, 0x6e, 0x65, 0x72, 0x67, 0x79,
	0x18, 0x06, 0x20, 0x01, 0x28, 0x01, 0x52, 0x06, 0x65, 0x6e, 0x65, 0x72, 0x67, 0x79, 0x12, 0x18,
	0x0a, 0x07, 0x63, 0x6f, 0x73, 0x74, 0x45, 0x55, 0x52, 0x18, 0x07, 0x20, 0x01, 0x28, 0x01, 0x52,
	0x07, 0x63, 0x6f, 0x73, 0x74, 0x45, 0x55, 0x52, 0x12, 0x22, 0x0a, 0x0c, 0x72, 0x65, 0x71, 0x75,
	0x69, 0x72, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x08, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0c,
	0x72, 0x65, 0x71, 0x75, 0x69, 0x72, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x22, 0x64, 0x0a, 0x10,
	0x50, 0x75, 0x72, 0x63, 0x68, 0x61, 0x73, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x16, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x12, 0x1e, 0x0a, 0x0a, 0x63, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x79,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0a, 0x63, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69,
	0x74, 0x79, 0x22, 0x86, 0x01, 0x0a, 0x06, 0x45, 0x6e, 0x72, 0x6f, 0x6c, 0x6c, 0x12, 0x12, 0x0a,
	0x04, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x4e, 0x61, 0x6d,
	0x65, 0x12, 0x18, 0x0a, 0x07, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x07, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x1a, 0x0a, 0x08, 0x4c,
	0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x03, 0x28, 0x01, 0x52, 0x08, 0x4c,
	0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x14, 0x0a, 0x05, 0x43, 0x6f, 0x73, 0x74, 0x48,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x01, 0x52, 0x05, 0x43, 0x6f, 0x73, 0x74, 0x48, 0x12, 0x1c, 0x0a,
	0x09, 0x4f, 0x66, 0x66, 0x65, 0x72, 0x69, 0x6e, 0x67, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x09, 0x4f, 0x66, 0x66, 0x65, 0x72, 0x69, 0x6e, 0x67, 0x73, 0x22, 0x42, 0x0a, 0x0e, 0x45,
	0x6e, 0x72, 0x6f, 0x6c, 0x6c, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x16, 0x0a,
	0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x22,
	0x41, 0x0a, 0x13, 0x4f, 0x70, 0x74, 0x69, 0x6d, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2a, 0x0a, 0x11, 0x6a, 0x73, 0x6f, 0x6e, 0x5f, 0x70,
	0x72, 0x6f, 0x62, 0x6c, 0x65, 0x6d, 0x5f, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0f, 0x6a, 0x73, 0x6f, 0x6e, 0x50, 0x72, 0x6f, 0x62, 0x6c, 0x65, 0x6d, 0x44, 0x61,
	0x74, 0x61, 0x22, 0x68, 0x0a, 0x14, 0x4f, 0x70, 0x74, 0x69, 0x6d, 0x69, 0x7a, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x2b, 0x0a, 0x09, 0x73, 0x6f,
	0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0d, 0x2e,
	0x44, 0x53, 0x4d, 0x2e, 0x53, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x09, 0x73, 0x6f,
	0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x23, 0x0a, 0x0d, 0x65, 0x72, 0x72, 0x6f, 0x72,
	0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c,
	0x65, 0x72, 0x72, 0x6f, 0x72, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x22, 0x9e, 0x01, 0x0a,
	0x08, 0x53, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x21, 0x0a, 0x0c, 0x75, 0x73, 0x65,
	0x72, 0x5f, 0x69, 0x6e, 0x64, 0x69, 0x63, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x05, 0x52,
	0x0b, 0x75, 0x73, 0x65, 0x72, 0x49, 0x6e, 0x64, 0x69, 0x63, 0x65, 0x73, 0x12, 0x19, 0x0a, 0x08,
	0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07,
	0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x73, 0x12, 0x25, 0x0a, 0x0e, 0x74, 0x72, 0x61, 0x6e, 0x73,
	0x70, 0x6f, 0x72, 0x74, 0x5f, 0x63, 0x6f, 0x73, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x01, 0x52,
	0x0d, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x70, 0x6f, 0x72, 0x74, 0x43, 0x6f, 0x73, 0x74, 0x12, 0x2d,
	0x0a, 0x12, 0x6d, 0x61, 0x6e, 0x75, 0x66, 0x61, 0x63, 0x74, 0x75, 0x72, 0x69, 0x6e, 0x67, 0x5f,
	0x63, 0x6f, 0x73, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x01, 0x52, 0x11, 0x6d, 0x61, 0x6e, 0x75,
	0x66, 0x61, 0x63, 0x74, 0x75, 0x72, 0x69, 0x6e, 0x67, 0x43, 0x6f, 0x73, 0x74, 0x32, 0xfd, 0x01,
	0x0a, 0x11, 0x53, 0x75, 0x62, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x53, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x12, 0x38, 0x0a, 0x10, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x41, 0x76, 0x61, 0x69,
	0x6c, 0x61, 0x62, 0x69, 0x6c, 0x74, 0x79, 0x12, 0x0c, 0x2e, 0x44, 0x53, 0x4d, 0x2e, 0x50, 0x72,
	0x6f, 0x63, 0x65, 0x73, 0x73, 0x1a, 0x14, 0x2e, 0x44, 0x53, 0x4d, 0x2e, 0x50, 0x72, 0x6f, 0x63,
	0x65, 0x73, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x37, 0x0a,
	0x0d, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x65, 0x73, 0x74, 0x12, 0x0d,
	0x2e, 0x44, 0x53, 0x4d, 0x2e, 0x50, 0x75, 0x72, 0x63, 0x68, 0x61, 0x73, 0x65, 0x1a, 0x15, 0x2e,
	0x44, 0x53, 0x4d, 0x2e, 0x50, 0x75, 0x72, 0x63, 0x68, 0x61, 0x73, 0x65, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x32, 0x0a, 0x0c, 0x45, 0x6e, 0x72, 0x6f, 0x6c, 0x6c,
	0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x12, 0x0b, 0x2e, 0x44, 0x53, 0x4d, 0x2e, 0x45, 0x6e, 0x72,
	0x6f, 0x6c, 0x6c, 0x1a, 0x13, 0x2e, 0x44, 0x53, 0x4d, 0x2e, 0x45, 0x6e, 0x72, 0x6f, 0x6c, 0x6c,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x41, 0x0a, 0x08, 0x4f, 0x70,
	0x74, 0x69, 0x6d, 0x69, 0x7a, 0x65, 0x12, 0x18, 0x2e, 0x44, 0x53, 0x4d, 0x2e, 0x4f, 0x70, 0x74,
	0x69, 0x6d, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x19, 0x2e, 0x44, 0x53, 0x4d, 0x2e, 0x4f, 0x70, 0x74, 0x69, 0x6d, 0x69, 0x7a, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x31, 0x5a,
	0x2f, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x45, 0x72, 0x69, 0x63,
	0x43, 0x68, 0x69, 0x71, 0x75, 0x69, 0x74, 0x6f, 0x47, 0x2f, 0x52, 0x65, 0x6d, 0x61, 0x6e, 0x65,
	0x74, 0x5f, 0x44, 0x53, 0x4d, 0x2f, 0x44, 0x53, 0x4d, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_DSM_proto_rawDescOnce sync.Once
	file_DSM_proto_rawDescData = file_DSM_proto_rawDesc
)

func file_DSM_proto_rawDescGZIP() []byte {
	file_DSM_proto_rawDescOnce.Do(func() {
		file_DSM_proto_rawDescData = protoimpl.X.CompressGZIP(file_DSM_proto_rawDescData)
	})
	return file_DSM_proto_rawDescData
}

var file_DSM_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_DSM_proto_goTypes = []any{
	(*Process)(nil),               // 0: DSM.Process
	(*ProcessResponse)(nil),       // 1: DSM.ProcessResponse
	(*Purchase)(nil),              // 2: DSM.Purchase
	(*PurchaseResponse)(nil),      // 3: DSM.PurchaseResponse
	(*Enroll)(nil),                // 4: DSM.Enroll
	(*EnrollResponse)(nil),        // 5: DSM.EnrollResponse
	(*OptimizationRequest)(nil),   // 6: DSM.OptimizationRequest
	(*OptimizationResponse)(nil),  // 7: DSM.OptimizationResponse
	(*Solution)(nil),              // 8: DSM.Solution
	(*timestamppb.Timestamp)(nil), // 9: google.protobuf.Timestamp
}
var file_DSM_proto_depIdxs = []int32{
	9, // 0: DSM.Process.submitted_at:type_name -> google.protobuf.Timestamp
	8, // 1: DSM.OptimizationResponse.solutions:type_name -> DSM.Solution
	0, // 2: DSM.SubmissionService.CheckAvailabilty:input_type -> DSM.Process
	2, // 3: DSM.SubmissionService.CheckInterest:input_type -> DSM.Purchase
	4, // 4: DSM.SubmissionService.EnrollServer:input_type -> DSM.Enroll
	6, // 5: DSM.SubmissionService.Optimize:input_type -> DSM.OptimizationRequest
	1, // 6: DSM.SubmissionService.CheckAvailabilty:output_type -> DSM.ProcessResponse
	3, // 7: DSM.SubmissionService.CheckInterest:output_type -> DSM.PurchaseResponse
	5, // 8: DSM.SubmissionService.EnrollServer:output_type -> DSM.EnrollResponse
	7, // 9: DSM.SubmissionService.Optimize:output_type -> DSM.OptimizationResponse
	6, // [6:10] is the sub-list for method output_type
	2, // [2:6] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_DSM_proto_init() }
func file_DSM_proto_init() {
	if File_DSM_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_DSM_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_DSM_proto_goTypes,
		DependencyIndexes: file_DSM_proto_depIdxs,
		MessageInfos:      file_DSM_proto_msgTypes,
	}.Build()
	File_DSM_proto = out.File
	file_DSM_proto_rawDesc = nil
	file_DSM_proto_goTypes = nil
	file_DSM_proto_depIdxs = nil
}
