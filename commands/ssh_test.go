package commands_test

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/bosh-bootloader/commands"
	"github.com/cloudfoundry/bosh-bootloader/fakes"
	"github.com/cloudfoundry/bosh-bootloader/storage"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SSH", func() {
	var (
		ssh          commands.SSH
		sshCmd       *fakes.SSHCmd
		sshKeyGetter *fakes.FancySSHKeyGetter
		fileIO       *fakes.FileIO
		randomPort   *fakes.RandomPort
	)

	BeforeEach(func() {
		sshCmd = &fakes.SSHCmd{}
		sshKeyGetter = &fakes.FancySSHKeyGetter{}
		fileIO = &fakes.FileIO{}
		randomPort = &fakes.RandomPort{}

		ssh = commands.NewSSH(sshCmd, sshKeyGetter, fileIO, randomPort)
	})

	Describe("CheckFastFails", func() {
		It("checks the bbl state for the jumpbox url", func() {
			err := ssh.CheckFastFails([]string{""}, storage.State{Jumpbox: storage.Jumpbox{URL: "some-jumpbox"}})
			Expect(err).NotTo(HaveOccurred())
		})

		Context("where there is no jumpbox url", func() {
			It("returns an error", func() {
				err := ssh.CheckFastFails([]string{""}, storage.State{})
				Expect(err).To(MatchError("Invalid bbl state for bbl ssh."))
			})
		})
	})

	Describe("Execute", func() {
		var (
			jumpboxPrivateKeyPath string
			state                 storage.State
		)
		BeforeEach(func() {
			fileIO.TempDirCall.Returns.Name = "some-temp-dir"
			sshKeyGetter.JumpboxGetCall.Returns.PrivateKey = "jumpbox-private-key"
			jumpboxPrivateKeyPath = filepath.Join("some-temp-dir", "jumpbox-private-key")

			state = storage.State{
				Jumpbox: storage.Jumpbox{
					URL: "jumpboxURL:22",
				},
				BOSH: storage.BOSH{
					DirectorAddress: "https://directorURL:25",
				},
			}
		})

		Context("--director", func() {
			var directorPrivateKeyPath string

			BeforeEach(func() {
				sshKeyGetter.DirectorGetCall.Returns.PrivateKey = "director-private-key"
				directorPrivateKeyPath = filepath.Join("some-temp-dir", "director-private-key")
				randomPort.GetPortCall.Returns.Port = "60000"
			})

			It("calls ssh with appropriate arguments", func() {
				err := ssh.Execute([]string{"--director"}, state)
				Expect(err).NotTo(HaveOccurred())

				Expect(sshKeyGetter.JumpboxGetCall.CallCount).To(Equal(1))
				Expect(sshKeyGetter.DirectorGetCall.CallCount).To(Equal(1))

				Expect(fileIO.WriteFileCall.CallCount).To(Equal(2))
				Expect(fileIO.WriteFileCall.Receives).To(ConsistOf(
					fakes.WriteFileReceive{
						Filename: jumpboxPrivateKeyPath,
						Contents: []byte("jumpbox-private-key"),
						Mode:     os.FileMode(0600),
					},
					fakes.WriteFileReceive{
						Filename: directorPrivateKeyPath,
						Contents: []byte("director-private-key"),
						Mode:     os.FileMode(0600),
					},
				))

				Expect(sshCmd.RunCall.Receives[0].Args).To(ConsistOf(
					"-4 -D", "60000", "-fNC", "jumpbox@jumpboxURL", "-i", jumpboxPrivateKeyPath,
				))

				Expect(sshCmd.RunCall.Receives[1].Args).To(ConsistOf(
					"-o ProxyCommand=nc -x localhost:60000 %h %p", "-i", directorPrivateKeyPath, "jumpbox@directorURL",
				))
			})

			Context("when ssh key getter fails to get director key", func() {
				It("returns the error", func() {
					sshKeyGetter.DirectorGetCall.Returns.Error = errors.New("fig")

					err := ssh.Execute([]string{"--director"}, state)

					Expect(err).To(MatchError("Get director private key: fig"))
				})
			})

			Context("when fileio fails to create a temp dir", func() {
				It("returns the error", func() {
					fileIO.TempDirCall.Returns.Error = errors.New("date")

					err := ssh.Execute([]string{"--director"}, state)

					Expect(err).To(MatchError("Create temp directory: date"))
				})
			})

			Context("when fileio fails to create a temp dir", func() {
				It("contextualizes a failure to write the private key", func() {
					fileIO.WriteFileCall.Returns = []fakes.WriteFileReturn{{Error: errors.New("boisenberry")}}

					err := ssh.Execute([]string{"--director"}, state)

					Expect(err).To(MatchError("Write private key file: boisenberry"))
				})
			})

			Context("when random port fails to return a port", func() {
				It("returns the error", func() {
					randomPort.GetPortCall.Returns.Error = errors.New("prune")

					err := ssh.Execute([]string{"--director"}, state)

					Expect(err).To(MatchError("Open proxy port: prune"))
				})
			})

			Context("when the ssh command fails to open a tunnel to the jumpbox", func() {
				It("returns the error", func() {
					sshCmd.RunCall.Returns = []fakes.SSHRunReturn{fakes.SSHRunReturn{Error: errors.New("lignonberry")}}

					err := ssh.Execute([]string{"--director"}, state)

					Expect(err).To(MatchError("Open tunnel to jumpbox: lignonberry"))
				})
			})
		})

		Context("--jumpbox", func() {
			It("calls ssh with appropriate arguments", func() {
				err := ssh.Execute([]string{"--jumpbox"}, state)
				Expect(err).NotTo(HaveOccurred())

				Expect(sshKeyGetter.JumpboxGetCall.CallCount).To(Equal(1))

				Expect(fileIO.WriteFileCall.CallCount).To(Equal(1))
				Expect(fileIO.WriteFileCall.Receives[0].Filename).To(Equal(jumpboxPrivateKeyPath))
				Expect(fileIO.WriteFileCall.Receives[0].Contents).To(Equal([]byte("jumpbox-private-key")))
				Expect(fileIO.WriteFileCall.Receives[0].Mode).To(Equal(os.FileMode(0600)))

				Expect(sshCmd.RunCall.Receives[0].Args).To(ConsistOf(
					"-o StrictHostKeyChecking=no -o ServerAliveInterval=300", "jumpbox@jumpboxURL", "-i", jumpboxPrivateKeyPath,
				))
			})

			Context("when ssh key getter fails to get the jumpbox ssh private key", func() {
				It("returns the error", func() {
					sshKeyGetter.JumpboxGetCall.Returns.Error = errors.New("fig")

					err := ssh.Execute([]string{"--jumpbox"}, state)

					Expect(err).To(MatchError("Get jumpbox private key: fig"))
				})
			})

			Context("when fileio fails to write the jumpbox private key", func() {
				It("returns the error", func() {
					fileIO.WriteFileCall.Returns = []fakes.WriteFileReturn{{Error: errors.New("boisenberry")}}

					err := ssh.Execute([]string{"--jumpbox"}, state)

					Expect(err).To(MatchError("Write private key file: boisenberry"))
				})
			})
		})

		Context("when the user does not provide a flag", func() {
			It("returns an error", func() {
				err := ssh.Execute([]string{}, storage.State{})
				Expect(err).To(MatchError("This command requires the --jumpbox or --director flag."))
			})
		})

		Context("when the user provides invalid flags", func() {
			It("returns an error", func() {
				err := ssh.Execute([]string{"--bogus-flag"}, storage.State{})
				Expect(err).To(MatchError("flag provided but not defined: -bogus-flag"))
			})
		})
	})
})
