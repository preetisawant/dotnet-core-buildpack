package brats_test

import (
	"github.com/blang/semver"
	"github.com/cloudfoundry/libbuildpack/bratshelper"
	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dotnet buildpack", func() {
	//bratshelper.UnbuiltBuildpack("dotnet", CopyBrats)
	//
	//bratshelper.DeployingAnAppWithAnUpdatedVersionOfTheSameBuildpack(CopyBrats)
	//
	//bratshelper.StagingWithADepThatIsNotTheLatestConstrained(
	//	"dotnet",
	//	FirstOfVersionLine("dotnet", "2.1.301"),
	//	func(v string) *cutlass.App { return CopyBratsWithFramework(v, "2.1.x") },
	//)
	//
	//bratshelper.StagingWithCustomBuildpackWithCredentialsInDependencies(
	//	`dotnet\.[\d\.]+\.linux\-amd64\-.*\-[\da-f]+\.tar.xz`,
	//	CopyBrats,
	//)
	//
	//bratshelper.DeployAppWithExecutableProfileScript("dotnet", CopyBrats)
	//
	//bratshelper.DeployAnAppWithSensitiveEnvironmentVariables(CopyBrats)

	compatible := func(sdkVersion, frameworkVersion string) bool {
		sdk, err := semver.Parse(sdkVersion)
		if err != nil {
			panic(err)
		}

		framework, err := semver.Parse(frameworkVersion)
		if err != nil {
			panic(err)
		}

		isCompatible := sdk.Major == framework.Major

		framework210 := semver.MustParse("2.1.0")
		if framework.GTE(framework210) {
			sdk21300 := semver.MustParse("2.1.300")
			isCompatible = isCompatible && sdk.GTE(sdk21300)
		}

		return isCompatible
	}

	ensureAppWorks := func(sdkVersion, frameworkVersion string, app *cutlass.App) {
		PushApp(app)

		By("installs the correct version of .NET SDK + .NET Framework", func() {
			Expect(app.Stdout.String()).To(ContainSubstring("Installing dotnet " + sdkVersion))
			Expect(app.Stdout.String()).To(MatchRegexp(
				"(Using dotnet framework installed in .*\\Q/dotnet/shared/Microsoft.NETCore.App/%s\\E|\\QInstalling dotnet-framework %s\\E)",
				frameworkVersion,
				frameworkVersion,
			))
		})

		By("runs a simple web server", func() {
			Expect(app.GetBody("/")).To(ContainSubstring("Hello World!"))
		})
	}

	//// For C# apps
	//bratshelper.ForAllSupportedVersions2(
	//	"dotnet",
	//	"dotnet-framework",
	//	compatible,
	//	"with .NET SDK version: %s and .NET Framework version: %s",
	//	CopyBratsWithFramework,
	//	ensureAppWorks,
	//)

	// For F# apps
	bratshelper.ForAllSupportedVersions2(
		"dotnet",
		"dotnet-framework",
		compatible,
		"with .NET SDK version: %s and .NET Framework version: %s",
		CopyFSharpBratsWithFramework,
		ensureAppWorks,
	)
})
