package tools

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Tools Package Finalizer", func() {
	var finalizer = "github.com/symcn"

	DescribeTable("AddFinalizer", func(existing, expected []string) {
		meta := &metav1.ObjectMeta{
			Finalizers: existing,
		}
		AddFinalizer(meta, finalizer)
		Expect(meta.Finalizers).To(Equal(expected))
	},
		Entry("add new finalizer", []string{"f1", "f2"}, []string{"f1", "f2", finalizer}),
		Entry("add old finalizer", []string{"f1", "f2", finalizer}, []string{"f1", "f2", finalizer}),
	)

	DescribeTable("HasFinalizer", func(existing []string, expected bool) {
		meta := &metav1.ObjectMeta{
			Finalizers: existing,
		}
		Expect(HasFinalizer(meta, finalizer)).To(Equal(expected))
	},
		Entry("exist", []string{"f1", finalizer}, true),
		Entry("not exist", []string{"f1", "f2"}, false),
	)

	DescribeTable("RemoveFinalizer", func(existing, expected []string) {
		meta := &metav1.ObjectMeta{
			Finalizers: existing,
		}
		RemoveFinalizer(meta, finalizer)
		Expect(meta.Finalizers).To(Equal(expected))
	},
		Entry("exist", []string{"f1", finalizer}, []string{"f1"}),
		Entry("not exist", []string{"f1", "f2"}, []string{"f1", "f2"}),
	)
})
