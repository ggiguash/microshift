// Code generated by applyconfiguration-gen. DO NOT EDIT.

package internal

import (
	fmt "fmt"
	sync "sync"

	typed "sigs.k8s.io/structured-merge-diff/v4/typed"
)

func Parser() *typed.Parser {
	parserOnce.Do(func() {
		var err error
		parser, err = typed.NewParser(schemaYAML)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse schema: %v", err))
		}
	})
	return parser
}

var parserOnce sync.Once
var parser *typed.Parser
var schemaYAML = typed.YAMLObject(`types:
- name: com.github.openshift.api.template.v1.BrokerTemplateInstance
  map:
    fields:
    - name: apiVersion
      type:
        scalar: string
    - name: kind
      type:
        scalar: string
    - name: metadata
      type:
        namedType: io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta
      default: {}
    - name: spec
      type:
        namedType: com.github.openshift.api.template.v1.BrokerTemplateInstanceSpec
      default: {}
- name: com.github.openshift.api.template.v1.BrokerTemplateInstanceSpec
  map:
    fields:
    - name: bindingIDs
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: atomic
    - name: secret
      type:
        namedType: io.k8s.api.core.v1.ObjectReference
      default: {}
    - name: templateInstance
      type:
        namedType: io.k8s.api.core.v1.ObjectReference
      default: {}
- name: com.github.openshift.api.template.v1.Parameter
  map:
    fields:
    - name: description
      type:
        scalar: string
    - name: displayName
      type:
        scalar: string
    - name: from
      type:
        scalar: string
    - name: generate
      type:
        scalar: string
    - name: name
      type:
        scalar: string
      default: ""
    - name: required
      type:
        scalar: boolean
    - name: value
      type:
        scalar: string
- name: com.github.openshift.api.template.v1.Template
  map:
    fields:
    - name: apiVersion
      type:
        scalar: string
    - name: kind
      type:
        scalar: string
    - name: labels
      type:
        map:
          elementType:
            scalar: string
    - name: message
      type:
        scalar: string
    - name: metadata
      type:
        namedType: io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta
      default: {}
    - name: objects
      type:
        list:
          elementType:
            namedType: __untyped_atomic_
          elementRelationship: atomic
    - name: parameters
      type:
        list:
          elementType:
            namedType: com.github.openshift.api.template.v1.Parameter
          elementRelationship: atomic
- name: com.github.openshift.api.template.v1.TemplateInstance
  map:
    fields:
    - name: apiVersion
      type:
        scalar: string
    - name: kind
      type:
        scalar: string
    - name: metadata
      type:
        namedType: io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta
      default: {}
    - name: spec
      type:
        namedType: com.github.openshift.api.template.v1.TemplateInstanceSpec
      default: {}
    - name: status
      type:
        namedType: com.github.openshift.api.template.v1.TemplateInstanceStatus
      default: {}
- name: com.github.openshift.api.template.v1.TemplateInstanceCondition
  map:
    fields:
    - name: lastTransitionTime
      type:
        namedType: io.k8s.apimachinery.pkg.apis.meta.v1.Time
    - name: message
      type:
        scalar: string
      default: ""
    - name: reason
      type:
        scalar: string
      default: ""
    - name: status
      type:
        scalar: string
      default: ""
    - name: type
      type:
        scalar: string
      default: ""
- name: com.github.openshift.api.template.v1.TemplateInstanceObject
  map:
    fields:
    - name: ref
      type:
        namedType: io.k8s.api.core.v1.ObjectReference
      default: {}
- name: com.github.openshift.api.template.v1.TemplateInstanceRequester
  map:
    fields:
    - name: extra
      type:
        map:
          elementType:
            list:
              elementType:
                scalar: string
              elementRelationship: atomic
    - name: groups
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: atomic
    - name: uid
      type:
        scalar: string
    - name: username
      type:
        scalar: string
- name: com.github.openshift.api.template.v1.TemplateInstanceSpec
  map:
    fields:
    - name: requester
      type:
        namedType: com.github.openshift.api.template.v1.TemplateInstanceRequester
    - name: secret
      type:
        namedType: io.k8s.api.core.v1.LocalObjectReference
    - name: template
      type:
        namedType: com.github.openshift.api.template.v1.Template
      default: {}
- name: com.github.openshift.api.template.v1.TemplateInstanceStatus
  map:
    fields:
    - name: conditions
      type:
        list:
          elementType:
            namedType: com.github.openshift.api.template.v1.TemplateInstanceCondition
          elementRelationship: atomic
    - name: objects
      type:
        list:
          elementType:
            namedType: com.github.openshift.api.template.v1.TemplateInstanceObject
          elementRelationship: atomic
- name: io.k8s.api.core.v1.LocalObjectReference
  map:
    fields:
    - name: name
      type:
        scalar: string
      default: ""
    elementRelationship: atomic
- name: io.k8s.api.core.v1.ObjectReference
  map:
    fields:
    - name: apiVersion
      type:
        scalar: string
    - name: fieldPath
      type:
        scalar: string
    - name: kind
      type:
        scalar: string
    - name: name
      type:
        scalar: string
    - name: namespace
      type:
        scalar: string
    - name: resourceVersion
      type:
        scalar: string
    - name: uid
      type:
        scalar: string
    elementRelationship: atomic
- name: io.k8s.apimachinery.pkg.apis.meta.v1.FieldsV1
  map:
    elementType:
      scalar: untyped
      list:
        elementType:
          namedType: __untyped_atomic_
        elementRelationship: atomic
      map:
        elementType:
          namedType: __untyped_deduced_
        elementRelationship: separable
- name: io.k8s.apimachinery.pkg.apis.meta.v1.ManagedFieldsEntry
  map:
    fields:
    - name: apiVersion
      type:
        scalar: string
    - name: fieldsType
      type:
        scalar: string
    - name: fieldsV1
      type:
        namedType: io.k8s.apimachinery.pkg.apis.meta.v1.FieldsV1
    - name: manager
      type:
        scalar: string
    - name: operation
      type:
        scalar: string
    - name: subresource
      type:
        scalar: string
    - name: time
      type:
        namedType: io.k8s.apimachinery.pkg.apis.meta.v1.Time
- name: io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta
  map:
    fields:
    - name: annotations
      type:
        map:
          elementType:
            scalar: string
    - name: creationTimestamp
      type:
        namedType: io.k8s.apimachinery.pkg.apis.meta.v1.Time
    - name: deletionGracePeriodSeconds
      type:
        scalar: numeric
    - name: deletionTimestamp
      type:
        namedType: io.k8s.apimachinery.pkg.apis.meta.v1.Time
    - name: finalizers
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: associative
    - name: generateName
      type:
        scalar: string
    - name: generation
      type:
        scalar: numeric
    - name: labels
      type:
        map:
          elementType:
            scalar: string
    - name: managedFields
      type:
        list:
          elementType:
            namedType: io.k8s.apimachinery.pkg.apis.meta.v1.ManagedFieldsEntry
          elementRelationship: atomic
    - name: name
      type:
        scalar: string
    - name: namespace
      type:
        scalar: string
    - name: ownerReferences
      type:
        list:
          elementType:
            namedType: io.k8s.apimachinery.pkg.apis.meta.v1.OwnerReference
          elementRelationship: associative
          keys:
          - uid
    - name: resourceVersion
      type:
        scalar: string
    - name: selfLink
      type:
        scalar: string
    - name: uid
      type:
        scalar: string
- name: io.k8s.apimachinery.pkg.apis.meta.v1.OwnerReference
  map:
    fields:
    - name: apiVersion
      type:
        scalar: string
      default: ""
    - name: blockOwnerDeletion
      type:
        scalar: boolean
    - name: controller
      type:
        scalar: boolean
    - name: kind
      type:
        scalar: string
      default: ""
    - name: name
      type:
        scalar: string
      default: ""
    - name: uid
      type:
        scalar: string
      default: ""
    elementRelationship: atomic
- name: io.k8s.apimachinery.pkg.apis.meta.v1.Time
  scalar: untyped
- name: io.k8s.apimachinery.pkg.runtime.RawExtension
  map:
    elementType:
      scalar: untyped
      list:
        elementType:
          namedType: __untyped_atomic_
        elementRelationship: atomic
      map:
        elementType:
          namedType: __untyped_deduced_
        elementRelationship: separable
- name: __untyped_atomic_
  scalar: untyped
  list:
    elementType:
      namedType: __untyped_atomic_
    elementRelationship: atomic
  map:
    elementType:
      namedType: __untyped_atomic_
    elementRelationship: atomic
- name: __untyped_deduced_
  scalar: untyped
  list:
    elementType:
      namedType: __untyped_atomic_
    elementRelationship: atomic
  map:
    elementType:
      namedType: __untyped_deduced_
    elementRelationship: separable
`)
