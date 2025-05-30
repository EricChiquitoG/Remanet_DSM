# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# NO CHECKED-IN PROTOBUF GENCODE
# source: DSM.proto
# Protobuf Python Version: 5.29.0
"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(
    _runtime_version.Domain.PUBLIC,
    5,
    29,
    0,
    '',
    'DSM.proto'
)
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()


from google.protobuf import timestamp_pb2 as google_dot_protobuf_dot_timestamp__pb2


DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\tDSM.proto\x12\x03\x44SM\x1a\x1fgoogle/protobuf/timestamp.proto\"\x95\x01\n\x07Process\x12\x11\n\tstep_name\x18\x01 \x01(\t\x12\x14\n\x0cproduct_type\x18\x02 \x01(\t\x12\x19\n\x11\x65\x63onomic_operator\x18\x03 \x01(\t\x12\x30\n\x0csubmitted_at\x18\x04 \x01(\x0b\x32\x1a.google.protobuf.Timestamp\x12\x14\n\x0crequirements\x18\x05 \x03(\t\"F\n\x0fProcessResponse\x12\x0e\n\x06status\x18\x01 \x01(\t\x12\x0f\n\x07message\x18\x02 \x01(\t\x12\x12\n\ncapability\x18\x03 \x03(\t\"\x99\x01\n\x08Purchase\x12\x14\n\x0cproduct_type\x18\x01 \x01(\t\x12\x0e\n\x06\x61mount\x18\x02 \x01(\t\x12\x10\n\x08location\x18\x03 \x03(\x02\x12\x11\n\tlogistics\x18\x04 \x01(\x01\x12\x0b\n\x03\x63o2\x18\x05 \x01(\x01\x12\x0e\n\x06\x65nergy\x18\x06 \x01(\x01\x12\x0f\n\x07\x63ostEUR\x18\x07 \x01(\x01\x12\x14\n\x0crequirements\x18\x08 \x03(\t\"G\n\x10PurchaseResponse\x12\x0e\n\x06status\x18\x01 \x01(\t\x12\x0f\n\x07message\x18\x02 \x01(\t\x12\x12\n\ncapability\x18\x03 \x01(\x08\"[\n\x06\x45nroll\x12\x0c\n\x04Name\x18\x01 \x01(\t\x12\x0f\n\x07\x41\x64\x64ress\x18\x02 \x01(\t\x12\x10\n\x08Location\x18\x03 \x03(\x01\x12\r\n\x05\x43ostH\x18\x04 \x01(\x01\x12\x11\n\tOfferings\x18\x05 \x03(\t\"1\n\x0e\x45nrollResponse\x12\x0e\n\x06status\x18\x01 \x01(\t\x12\x0f\n\x07message\x18\x02 \x01(\t\"0\n\x13OptimizationRequest\x12\x19\n\x11json_problem_data\x18\x01 \x01(\t\"O\n\x14OptimizationResponse\x12 \n\tsolutions\x18\x01 \x03(\x0b\x32\r.DSM.Solution\x12\x15\n\rerror_message\x18\x02 \x01(\t\"f\n\x08Solution\x12\x14\n\x0cuser_indices\x18\x01 \x03(\x05\x12\x10\n\x08user_ids\x18\x02 \x03(\t\x12\x16\n\x0etransport_cost\x18\x03 \x01(\x01\x12\x1a\n\x12manufacturing_cost\x18\x04 \x01(\x01\x32\xfd\x01\n\x11SubmissionService\x12\x38\n\x10\x43heckAvailabilty\x12\x0c.DSM.Process\x1a\x14.DSM.ProcessResponse\"\x00\x12\x37\n\rCheckInterest\x12\r.DSM.Purchase\x1a\x15.DSM.PurchaseResponse\"\x00\x12\x32\n\x0c\x45nrollServer\x12\x0b.DSM.Enroll\x1a\x13.DSM.EnrollResponse\"\x00\x12\x41\n\x08Optimize\x12\x18.DSM.OptimizationRequest\x1a\x19.DSM.OptimizationResponse\"\x00\x42\x31Z/github.com/EricChiquitoG/Remanet_DSM/DSM_protosb\x06proto3')

_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'DSM_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
  _globals['DESCRIPTOR']._loaded_options = None
  _globals['DESCRIPTOR']._serialized_options = b'Z/github.com/EricChiquitoG/Remanet_DSM/DSM_protos'
  _globals['_PROCESS']._serialized_start=52
  _globals['_PROCESS']._serialized_end=201
  _globals['_PROCESSRESPONSE']._serialized_start=203
  _globals['_PROCESSRESPONSE']._serialized_end=273
  _globals['_PURCHASE']._serialized_start=276
  _globals['_PURCHASE']._serialized_end=429
  _globals['_PURCHASERESPONSE']._serialized_start=431
  _globals['_PURCHASERESPONSE']._serialized_end=502
  _globals['_ENROLL']._serialized_start=504
  _globals['_ENROLL']._serialized_end=595
  _globals['_ENROLLRESPONSE']._serialized_start=597
  _globals['_ENROLLRESPONSE']._serialized_end=646
  _globals['_OPTIMIZATIONREQUEST']._serialized_start=648
  _globals['_OPTIMIZATIONREQUEST']._serialized_end=696
  _globals['_OPTIMIZATIONRESPONSE']._serialized_start=698
  _globals['_OPTIMIZATIONRESPONSE']._serialized_end=777
  _globals['_SOLUTION']._serialized_start=779
  _globals['_SOLUTION']._serialized_end=881
  _globals['_SUBMISSIONSERVICE']._serialized_start=884
  _globals['_SUBMISSIONSERVICE']._serialized_end=1137
# @@protoc_insertion_point(module_scope)
