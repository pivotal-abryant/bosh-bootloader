package compute_test

import (
	"errors"

	"github.com/genevievelesperance/leftovers/gcp/compute"
	"github.com/genevievelesperance/leftovers/gcp/compute/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gcpcompute "google.golang.org/api/compute/v1"
)

var _ = Describe("Firewalls", func() {
	var (
		client *fakes.FirewallsClient
		logger *fakes.Logger

		firewalls compute.Firewalls
	)

	BeforeEach(func() {
		client = &fakes.FirewallsClient{}
		logger = &fakes.Logger{}

		firewalls = compute.NewFirewalls(client, logger)
	})

	Describe("List", func() {
		var filter string

		BeforeEach(func() {
			logger.PromptCall.Returns.Proceed = true
			client.ListFirewallsCall.Returns.Output = &gcpcompute.FirewallList{
				Items: []*gcpcompute.Firewall{{
					Name: "banana-firewall",
				}},
			}
			filter = "banana"
		})

		It("lists, filters, and prompts for firewalls to delete", func() {
			list, err := firewalls.List(filter)
			Expect(err).NotTo(HaveOccurred())

			Expect(client.ListFirewallsCall.CallCount).To(Equal(1))

			Expect(logger.PromptCall.Receives.Message).To(Equal("Are you sure you want to delete firewall banana-firewall?"))

			Expect(list).To(HaveLen(1))
			Expect(list).To(HaveKeyWithValue("banana-firewall", ""))
		})

		Context("when the client fails to list firewalls", func() {
			BeforeEach(func() {
				client.ListFirewallsCall.Returns.Error = errors.New("some error")
			})

			It("returns the error", func() {
				_, err := firewalls.List(filter)
				Expect(err).To(MatchError("Listing firewalls: some error"))
			})
		})

		Context("when the firewall name does not contain the filter", func() {
			It("does not add it to the list", func() {
				list, err := firewalls.List("grape")
				Expect(err).NotTo(HaveOccurred())

				Expect(logger.PromptCall.CallCount).To(Equal(0))
				Expect(list).To(HaveLen(0))
			})
		})

		Context("when the user says no to the prompt", func() {
			BeforeEach(func() {
				logger.PromptCall.Returns.Proceed = false
			})

			It("does not add it to the list", func() {
				list, err := firewalls.List(filter)
				Expect(err).NotTo(HaveOccurred())

				Expect(list).To(HaveLen(0))
			})
		})
	})

	Describe("Delete", func() {
		var list map[string]string

		BeforeEach(func() {
			list = map[string]string{"banana-firewall": ""}
		})

		It("deletes firewalls", func() {
			firewalls.Delete(list)

			Expect(client.DeleteFirewallCall.CallCount).To(Equal(1))
			Expect(client.DeleteFirewallCall.Receives.Firewall).To(Equal("banana-firewall"))

			Expect(logger.PrintfCall.Messages).To(Equal([]string{"SUCCESS deleting firewall banana-firewall\n"}))
		})

		Context("when the client fails to delete a firewall", func() {
			BeforeEach(func() {
				client.DeleteFirewallCall.Returns.Error = errors.New("some error")
			})

			It("logs the error", func() {
				firewalls.Delete(list)

				Expect(logger.PrintfCall.Messages).To(Equal([]string{"ERROR deleting firewall banana-firewall: some error\n"}))
			})
		})
	})
})