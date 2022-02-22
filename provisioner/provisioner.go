package provisioner

import "errors"

func GetLatestNodeVersion(nodeVersion string) (string, error) {
	if nodeVersion != "" {
		_, err := gitRelease("mysteriumnetwork", "node", nodeVersion)
		if err != nil {
			return "", err
		}
		return nodeVersion, nil
	}

	releases, err := GitReleases("mysteriumnetwork", "node", 1)
	if err != nil {
		return "", err
	}

	if len(releases) == 0 {
		return "", errors.New("did not find a single release of node")
	}
	return releases[0].TagName, nil
}
