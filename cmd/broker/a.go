package main

type A interface {
	A()
}

type AA interface {
	A()
}

type B interface {
	B()
}

type Impl struct{}

func (i *Impl) A() {

}

func (i *Impl) B() {

}

type Impl2 struct{}

func (i *Impl2) B() {

}
