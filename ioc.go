package ioc

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/codegangsta/inject"
)

// Lifecycle is singleton or transient
type Lifecycle int

const (
	// Singleton is single instance
	Singleton Lifecycle = 0

	// Transient is temporary instance
	Transient Lifecycle = 1
)

// Initializer is to init a struct.
type Initializer interface {
	InitFunc() interface{}
}

// Container is a container for ioc.
type Container interface {
	// Register is to register a type as singleton or transient.
	Register(val interface{}, lifecycle Lifecycle)

	// RegisterTo is to register a interface as singleton or transient.
	RegisterTo(val interface{}, ifacePtr interface{}, lifecycle Lifecycle)

	// Resolve is to get instance of type.
	Resolve(typ reflect.Type) reflect.Value

	// Invoke is to inject to function's params, such as construction.
	Invoke(f interface{}) ([]reflect.Value, error)

	// SetParent is to resolve parent's container if current hasn't registered a type.
	SetParent(parent Container)
}

// NewContainer : create iocContainer
func NewContainer() Container {
	return &iocContainer{locker: &sync.RWMutex{}, i: inject.New(), typemapper: make(map[reflect.Type]reflect.Type)}
}

type iocContainer struct {
	locker     *sync.RWMutex
	i          inject.Injector
	typemapper map[reflect.Type]reflect.Type
	parent     Container
}

func (container *iocContainer) Register(val interface{}, lifecycle Lifecycle) {
	defer container.locker.Unlock()
	container.locker.Lock()

	if container.i.Get(reflect.TypeOf(val)).IsValid() {
		lifecycle = Singleton
	} else if container.typemapper[reflect.TypeOf(val)] != nil {
		lifecycle = Transient
	}
	switch lifecycle {
	case Singleton:
		container.i.Map(val)
	case Transient:
		typ := reflect.TypeOf(val)
		container.typemapper[typ] = fromPtrType(typ)
	}
}

func (container *iocContainer) RegisterTo(val interface{}, ifacePtr interface{}, lifecycle Lifecycle) {
	defer container.locker.Unlock()
	container.locker.Lock()

	if container.i.Get(reflect.TypeOf(val)).IsValid() {
		lifecycle = Singleton
	} else if container.typemapper[reflect.TypeOf(val)] != nil {
		lifecycle = Transient
	}
	switch lifecycle {
	case Singleton:
		container.i.MapTo(val, ifacePtr)
	case Transient:
		typ := fromPtrTypeOf(ifacePtr)
		container.typemapper[typ] = fromPtrTypeOf(val)
	}
}

func (container *iocContainer) Resolve(typ reflect.Type) reflect.Value {
	defer container.locker.RUnlock()
	container.locker.RLock()

	if val := container.i.Get(typ); val.IsValid() {
		if initializer, ok := val.Interface().(Initializer); ok {
			container.Invoke(initializer.InitFunc())
		}
		return val
	} else if realTyp := container.typemapper[typ]; realTyp != nil {
		val := reflect.New(realTyp)
		if typ.Kind() != reflect.Ptr && typ.Kind() != reflect.Interface {
			val = val.Elem()
		}
		if initializer, ok := val.Interface().(Initializer); ok {
			container.Invoke(initializer.InitFunc())
		}
		return val
	} else if container.parent != nil {
		return container.parent.Resolve(typ)
	}
	return reflect.Value{}
}

func (container *iocContainer) Invoke(f interface{}) ([]reflect.Value, error) {
	t := reflect.TypeOf(f)
	var in = make([]reflect.Value, t.NumIn()) //Panic if t is not kind of Func
	for i := 0; i < t.NumIn(); i++ {
		argType := t.In(i)
		val := container.Resolve(argType)
		if !val.IsValid() {
			return nil, fmt.Errorf("Value not found for type %v", argType)
		}

		in[i] = val
	}
	return reflect.ValueOf(f).Call(in), nil
}

func (container *iocContainer) SetParent(parent Container) {
	container.parent = parent
}

func fromPtrTypeOf(obj interface{}) reflect.Type {
	realType := reflect.TypeOf(obj)
	for realType.Kind() == reflect.Ptr {
		realType = realType.Elem()
	}
	return realType
}

func fromPtrType(typ reflect.Type) reflect.Type {
	realType := typ
	for realType.Kind() == reflect.Ptr {
		realType = realType.Elem()
	}
	return realType
}
