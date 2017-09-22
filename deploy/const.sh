# project='eng-infrastructure'
project='eng-infrastructure'
registry="gcr.io/$project"
repoDir=$(cd $(dirname $BASH_SOURCE)/..; pwd)
repo=$(basename $repoDir)
app=$repo

# Create the build tag
tag=$(cd $repoDir; git rev-parse HEAD | cut -c 1-10)
# tag=$(python $repoDir/drugdiscovery/_version.py)
# If there are uncommited changes, tag to indicate that this is snapshot after the last commit,
# i.e. don't know what's in the image
if [[ -n $(cd $repoDir; git status -sz) ]]; then
	tag=$tag-snapshot
fi

deployImage=$app:$tag

gcloud config set core/project $project
