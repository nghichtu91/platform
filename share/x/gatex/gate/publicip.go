package gate

import "github.com/nghichtu91/platform/share/planx/iputil"

func AwsGetPublicIP() (string, error) {
	return iputil.AwsGetPublicIP()
}

func externalIP() (string, error) {
	return iputil.ExternalIP()
}

func GetPublicIP(pip, listen string) string {
	return iputil.GetPublicIP(pip, listen)

}
