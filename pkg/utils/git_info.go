package utils

func GetGitInfo() (string, string, error) {
	if !IsGitRepository() {
		return "", "", nil
	}
	
	repo, repoErr := GetRepositoryInfo()
	branch, branchErr := GetCurrentBranch()
	
	if repoErr != nil && branchErr != nil {
		if repoErr != nil {
			return "", "", repoErr
		}
		return "", "", branchErr
	}
	
	return repo, branch, nil
}