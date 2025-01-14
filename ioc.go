// Package ioc is Inversion of Control (IoC).
// Support singleton and transient.
//
// The MIT License (MIT)
//
// # Copyright (c) 2016 Jerry Bai
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package ioc

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

const DefaultInitializeMethodName string = "Initialize"

// CustomInitializer that use customize initialize method instead of default method 'Initialize'
type CustomInitializer interface {
	// InitializeMethodName indicate the initialize method name.
	// Won't be invoked if the method returns is not exists.
	InitializeMethodName() string
}

var globalContainer Container = New()
var resolverType reflect.Type = reflect.TypeOf((*Resolver)(nil)).Elem()

// New ioc container, and add singleton service 'ioc.Resolver' to it.
func New() Container {
	var c Container = &defaultContainer{}
	c.AddSingleton(resolverType, c)
	return c
}

// Inversion of Control container.
type Container interface {
	Resolver

	// AddSingleton to add singleton instance.
	//
	//  // service
	//  type Service1 interface {
	//      Method1()
	//  }
	//  // implementation of service
	//  type ServiceImplementation1 struct {
	//      Field1 string
	//  }
	//  func(si *ServiceImplementation1) Method1() {}
	//	func(si *ServiceImplementation1) Initialize(resolver ioc.Resolver) {
	//	    si.resolver = resolver
	//	}
	//
	//  var container ioc.Container
	//  // interface as service, register as singleton
	//  err := container.AddSingleton(reflect.TypeOf((*Service1)(nil)).Elem(), &ServiceImplementation1{Field1: "abc"})
	//  // or *struct as service, register as singleton
	//  err = container.AddSingleton(reflect.TypeOf((*ServiceImplementation1)(nil)), &ServiceImplementation1{Field1: "abc"})
	AddSingleton(serviceType reflect.Type, instance any) error

	// AddTransient to add transient by instance factory.
	//
	//  // service
	//  type Service1 interface {
	//      Method1()
	//  }
	//  // implementation of service
	//  type ServiceImplementation1 struct {
	//      Field1 string
	//  }
	//  func(si *ServiceImplementation1) Method1() {}
	//
	//  var container ioc.Container
	//  // interface as service, register as transient
	//  err = container.AddTransient(reflect.TypeOf((*Service1)(nil)).Elem(), func() any {
	//      return &ServiceImplementation1{Field1: "abc"}
	//  })
	//  // or *struct as service, register as transient
	//  err = container.AddTransient(reflect.TypeOf((*ServiceImplementation1)(nil)), func() any {
	//      return &ServiceImplementation1{Field1: "abc"}
	//  })
	AddTransient(serviceType reflect.Type, instanceFactory func() any) error
}

// Resolver can resolve service.
type Resolver interface {
	// Set parent resolver, for resolving from parent if service not found in current.
	SetParent(parent Resolver)

	// Resolve to get service.
	//
	//  // service
	//  type Service1 interface {
	//      Method1()
	//  }
	//  // implementation of service
	//  type ServiceImplementation1 struct {
	//      Field1 string
	//  }
	//  func(si *ServiceImplementation1) Method1() {}
	//
	//  var container ioc.Container
	//  // interface as service
	//  service1 := container.Resolve(reflect.TypeOf((*Service1)(nil)).Elem())
	//  // or *struct as service
	//  service2 := container.Resolve(reflect.TypeOf((*ServiceImplementation1)(nil)))
	Resolve(serviceType reflect.Type) reflect.Value
}

// AddSingleton to add singleton instance.
//
// It will panic if 'TService' or 'instance' is invalid.
//
//	// service
//	type Service1 interface {
//	    Method1()
//	}
//	// implementation of service
//	type ServiceImplementation1 struct {
//	    Field1 string
//
//	    resolver ioc.Resolver
//	}
//	func(si *ServiceImplementation1) Method1() {}
//	func(si *ServiceImplementation1) Initialize(resolver ioc.Resolver) {
//	    si.resolver = resolver
//	}
//
//	// interface as service
//	ioc.AddSingleton[Service1](&ServiceImplementation1{Field1: "abc"})
//	// or *struct as service
//	ioc.AddSingleton[*ServiceImplementation1](&ServiceImplementation1{Field1: "abc"})
func AddSingleton[TService any](instance TService) {
	AddSingletonToC[TService](globalContainer, instance)
}

// AddSingletonToC to add singleton instance to container.
//
// It will panic if 'TService' or 'instance' is invalid.
func AddSingletonToC[TService any](container Container, instance TService) {
	err := container.AddSingleton(reflect.TypeOf((*TService)(nil)).Elem(), instance)
	if err != nil {
		panic(err)
	}
	getFieldsToInject(reflect.ValueOf(instance).Type())
}

// AddTransient to add transient service instance factory.
//
// It will panic if 'TService' or 'instance' is invalid.
//
//	// service
//	type Service1 interface {
//	    Method1()
//	}
//	// implementation of service
//	type ServiceImplementation1 struct {
//	    Field1 string
//	}
//	func(si *ServiceImplementation1) Method1() {}
//
//	// interface as service
//	ioc.AddTransient[Service1](func() Service1 {
//	     return &ServiceImplementation1{Field1: "abc"}
//	})
//	// or *struct as service
//	ioc.AddTransient[*ServiceImplementation1](func() *ServiceImplementation1 {
//	     return &ServiceImplementation1{Field1: "abc"}
//	})
func AddTransient[TService any](instanceFactory func() TService) {
	AddTransientToC[TService](globalContainer, instanceFactory)
}

// AddTransientToC to add transient service instance factory to container.
//
// It will panic if 'TService' or 'instance' is invalid.
func AddTransientToC[TService any](container Container, instanceFactory func() TService) {
	if instanceFactory == nil {
		panic("param 'instanceFactory' is null")
	}
	err := container.AddTransient(reflect.TypeOf((*TService)(nil)).Elem(), func() any {
		return instanceFactory()
	})
	if err != nil {
		panic(err)
	}
}

// GetService to get service.
//
//	// service
//	type Service1 interface {
//	    Method1()
//	}
//	// implementation of service
//	type ServiceImplementation1 struct {
//	    Field1 string
//	}
//	func(si *ServiceImplementation1) Method1() {}
//
//	// interface as service
//	service1 := ioc.GetService[Service1]()
//	// or *struct as service
//	service2 := ioc.GetService[*ServiceImplementation1]()
func GetService[TService any]() TService {
	return GetServiceFromC[TService](globalContainer)
}

// GetServiceFromC to get service from container.
func GetServiceFromC[TService any](container Container) TService {
	var instance TService
	instanceVal := container.Resolve(reflect.TypeOf((*TService)(nil)).Elem())
	if !instanceVal.IsValid() {
		return instance
	}
	instanceInterface := instanceVal.Interface()
	if instanceInterface != nil {
		if val, ok := instanceInterface.(TService); ok {
			instance = val
		}
	}
	return instance
}

// Inject to func or *struct with service.
// Field with type 'ioc.Resolver', will always been injected.
//
//	// service
//	type Service1 interface {
//	    Method1()
//	}
//
//	// implementation of service
//	type ServiceImplementation1 struct {
//	    Field1 string
//	}
//	func(si *ServiceImplementation1) Method1() {}
//
//	// client
//	type Client struct {
//	    Field1 Service1 `ioc-inject:"true"`
//	    Field2 *ServiceImplementation1 `ioc-inject:"true"`
//	}
//	func(c *Client) Method1(p1 Service1, p2 *ServiceImplementation1) {
//	    c.Field1 = p1
//	    c.Field2 = p2
//	}
//
//	var c client
//	// inject to func
//	ioc.Inject(c.Method1)
//	// inject to *struct
//	ioc.Inject(&c)
func Inject(target any) {
	InjectFromC(globalContainer, target)
}

// InjectFromC to inject to func or *struct or their's reflect.Value with service from container.
// Field with type 'ioc.Resolver', will always been injected.
func InjectFromC(container Container, target any) {
	var targetVal reflect.Value
	if val, ok := target.(reflect.Value); ok {
		targetVal = val
	} else {
		targetVal = reflect.ValueOf(target)
	}
	if !targetVal.IsValid() || targetVal.IsZero() {
		return
	}
	targetType := targetVal.Type()
	if targetType.Kind() == reflect.Func {
		// inject to func
		var in = make([]reflect.Value, targetType.NumIn())
		for i := 0; i < targetType.NumIn(); i++ {
			argType := targetType.In(i)
			val := container.Resolve(argType)
			if !val.IsValid() {
				in[i] = reflect.Zero(argType)
			} else {
				in[i] = val
			}
		}
		targetVal.Call(in)
	} else if targetType.Kind() == reflect.Pointer && targetType.Elem().Kind() == reflect.Struct {
		// skip implementation of ioc.Resolver
		if targetType.Implements(resolverType) {
			return
		}

		// inject to *struct
		structType := targetType.Elem()
		fields := getFieldsToInject(structType)
		for _, field := range fields {
			fieldVal := targetVal.Elem().Field(field.FieldIndex)
			val := container.Resolve(field.FieldType)
			if val.IsValid() {
				fieldVal.Set(val)
			}
		}
	}
}

// Set parent resolver, for resolving from parent if service not found in current.
func SetParent(parent Resolver) {
	globalContainer.SetParent(parent)
}

var structTypeToFieldsCache sync.Map

func getFieldsToInject(targetType reflect.Type) []structField {
	structType := targetType
	for structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
	}
	if structType.Kind() != reflect.Struct {
		return nil
	}

	if val, ok := structTypeToFieldsCache.Load(structType); ok {
		return val.([]structField)
	}
	fields := make([]structField, 0, structType.NumField())
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if !field.IsExported() || field.Anonymous {
			continue
		}
		canInject := field.Type == resolverType
		if !canInject {
			if val, ok := field.Tag.Lookup("ioc-inject"); ok && val == "true" {
				canInject = true
			}
		}
		if canInject {
			fields = append(fields, structField{
				FieldIndex: i,
				FieldType:  field.Type,
			})
		}
	}
	structTypeToFieldsCache.Store(structType, fields)
	return fields
}

type structField struct {
	FieldIndex int
	FieldType  reflect.Type
}

var _ Container = (*defaultContainer)(nil)

type defaultContainer struct {
	bindings sync.Map
	parent   Resolver
	locker   sync.Mutex
}

func (c *defaultContainer) Resolve(serviceType reflect.Type) reflect.Value {
	binding := c.getBinding(serviceType)
	if binding != nil {
		if binding.Instance.IsValid() {
			if !binding.InstanceInitialized {
				defer binding.Unlock()
				binding.Lock()
				Inject(binding.Instance)
				if binding.InstanceInitializer.IsValid() {
					func() {
						defer recover()
						Inject(binding.InstanceInitializer)
					}()
				}
				binding.InstanceInitialized = true
			}
			return binding.Instance
		}
		return reflect.ValueOf(binding.InstanceFactory())
	} else {
		parent := c.parent
		if parent != nil {
			return parent.Resolve(serviceType)
		} else {
			return reflect.Value{}
		}
	}
}

func (c *defaultContainer) SetParent(parent Resolver) {
	defer c.locker.Unlock()
	c.locker.Lock()
	if parent == nil || c.parent == parent {
		return
	}

	if c.parent == nil {
		c.parent = parent
	} else {
		c.parent.SetParent(parent)
	}
}

func (c *defaultContainer) AddSingleton(serviceType reflect.Type, instance any) error {
	if serviceType == nil {
		return errors.New("param 'serviceType' is null")
	}
	if instance == nil || reflect.ValueOf(instance).IsZero() {
		return errors.New("param 'instance' is null")
	}
	binding := c.getBinding(serviceType)
	if binding != nil {
		// ignore exists service in current container
		return nil
	}
	binding = &serviceBinding{ServiceType: serviceType, Instance: reflect.ValueOf(instance)}
	if serviceType != resolverType {
		initializeMethodName := DefaultInitializeMethodName
		if initializer, ok := binding.Instance.Interface().(CustomInitializer); ok {
			initializeMethodName = initializer.InitializeMethodName()
		}
		if foundMethod := binding.Instance.MethodByName(initializeMethodName); foundMethod.IsValid() {
			methodType := foundMethod.Type()
			for i := 0; i < methodType.NumIn(); i++ {
				if methodType.In(i) == serviceType {
					return fmt.Errorf("cycle reference: param[%d]'s type in method '%s' equals to service '%v'", i, initializeMethodName, serviceType)
				}
			}
			binding.InstanceInitializer = foundMethod
		}
	}
	return c.addBinding(binding)
}

func (c *defaultContainer) AddTransient(serviceType reflect.Type, instanceFactory func() any) error {
	if serviceType == nil {
		return errors.New("param 'serviceType' is null")
	}
	if instanceFactory == nil {
		return errors.New("param 'instanceFactory' is null")
	}
	binding := c.getBinding(serviceType)
	if binding != nil {
		// ignore exists service in current container
		return nil
	}
	binding = &serviceBinding{ServiceType: serviceType, InstanceFactory: instanceFactory}
	return c.addBinding(binding)
}

func (c *defaultContainer) addBinding(binding *serviceBinding) error {
	if binding != nil && binding.ServiceType != nil {
		if binding.ServiceType.Kind() != reflect.Interface &&
			!(binding.ServiceType.Kind() == reflect.Pointer && binding.ServiceType.Elem().Kind() == reflect.Struct) {
			return fmt.Errorf("type of service '%v' should be an interface or *struct", binding.ServiceType)
		}
		if binding.Instance.IsValid() {
			if !binding.Instance.Type().AssignableTo(binding.ServiceType) {
				return fmt.Errorf("instance should implement the service '%v'", binding.ServiceType)
			}
		}
		c.bindings.LoadOrStore(binding.ServiceType, binding)
	}
	return nil
}

func (c *defaultContainer) getBinding(serviceType reflect.Type) *serviceBinding {
	if bindingVal, ok := c.bindings.Load(serviceType); ok {
		binding := bindingVal.(*serviceBinding)
		return binding
	}
	return nil
}

type serviceBinding struct {
	ServiceType         reflect.Type
	Instance            reflect.Value
	InstanceInitializer reflect.Value
	InstanceInitialized bool
	InstanceFactory     func() any

	initializerLocker sync.Mutex
}

func (b *serviceBinding) Lock() {
	b.initializerLocker.Lock()
}

func (b *serviceBinding) Unlock() {
	b.initializerLocker.Unlock()
}
