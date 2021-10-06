#!/bin/bash
# Help function to show how this works
function full_help {
  cat << "TXT"
Infrastructure Deployment tool for deploying CloudFormation stacks.

Based on the fog.sh helper script from https://github.com/ArjenSchwarz/fog

It does so by attempting to create a changeset, showing a summary of this, and
then allowing you to deploy the changeset. Assuming the changeset creation was
successful.

After you deploy the changeset, it will wait for the deployment to complete. If
successful, the stack's Outputs will be shown, and if it fails it will show the
error messages and display a link to the full events list.

Additional functionalities:
- Verifies that production deployments happen from the master branch (commented out)
- If a new stack is created and fails to do so, offers to delete the stack
- Deletes a ChangeSet if it's decided not to deploy it unless -q is used

Available flags:

-n) *required* The name of the CloudFormation stack
-f) *required* The template file: e.g. `-f s3-buckets` for a template stored in
    templates/s3-buckets.yml. The templates directory and extension are
    automatically added. You can use subdirectories with e.g.
    `-f permission-sets/poweruser` to use the file
    templates/permission-sets/poweruser.yml
-d) Override the template path from templates
-t) *required* The JSON file containing the tags to be applied to the stack.
    e.g. production when you wish to use tags/production.json
-e) An optional additional tag file. Full relative path and filename required to
    be provided. Will be merged with the main tag file.
-m) A JSON string of tags that need to be applied. e.g.
    `-m '[{"Key": "SourceControl:TemplatePath","Value": "src/cloudformation/general/templates/simple-vpc.yml"}]'`.
    Will be merged with the main tag file.
-p) The JSON file containing the parameters to be applied to the stack. e.g.
    `-p s3-buckets-prod` when you wish to use parameters/s3-buckets-prod.json
-c) An optional additional parameter file. Full relative path and filename
    required to be provided. Will be merged with the main parameters file.
-o) A JSON string of parameters that need to be applied. e.g.
    `-o '[{"ParameterKey": "EnvType","ParameterValue": "prod"}]'`. Will be
    merged with the main parameters file.
-s) The name of the bucket you wish to upload the template to. Needs only to be
    used when the provided template is too large to deploy directly (as per
    CloudFormation limitations)
-q) Flag that only creates a ChangeSet instead of also deploying it. Generally
    used in CICD environments. The provided value is the name of the to be
    created ChangeSet. e.g. `-q Jenkins-12`
-y) Flag that will only deploy the provided ChangeSet. e.g. `-y Jenkins-12`
-a) Flag that will create and immediately deploy a ChangeSet
-x) Flag that will DELETE the provided ChangeSet. `e.g. -x Jenkins-12`
-h) Display this help

Basic example usage:
$ infra-deploy.sh -n VPC-Sandbox -f simple-vpc -t sandbox-commontags -p sandbox-vpc

Jenkins usage:
To create ChangeSet:
$ infra-deploy.sh -n VPC-Sandbox -f simple-vpc -t sandbox-commontags -p sandbox-vpc -q Jenkins-${BUILD_NUMBER}
After approval:
$ infra-deploy.sh -n VPC-Sandbox -f simple-vpc -t sandbox-commontags -p sandbox-vpc -y Jenkins-${BUILD_NUMBER}

TXT
}

MINIMUM_FOG_VERSION="0.8.0"
ALLOW_DEV_FOG=true
# TODO: clean up temporary tag files
TEMPLATE_OVERRIDE_PATH=
while getopts "n:f:t:e:m:p:o:c:d:s:q:y:x:a:h" opt; do
  case $opt in
    d)
      TEMPLATE_OVERRIDE_PATH=${OPTARG}
      ;;
    n)
      NAME=$OPTARG
      ;;
    f)
      DEFAULT_SUFFIX="yml"
      FILE_SUFFIX=$DEFAULT_SUFFIX
      RAW_TEMPLATE_FILE="${OPTARG}"
      ALTERNATIVE_SUFFIX="yaml"
      if [[ -n ${TEMPLATE_OVERRIDE_PATH} ]]; then
        if [[ ! -f "${TEMPLATE_OVERRIDE_PATH}/${OPTARG}.${DEFAULT_SUFFIX}" ]]; then
          if [[ -f "${TEMPLATE_OVERRIDE_PATH}/${OPTARG}.${ALTERNATIVE_SUFFIX}" ]]; then
            FILE_SUFFIX=$ALTERNATIVE_SUFFIX
          fi
        fi
        TEMPLATE_FILE=("--template-body" "file://${TEMPLATE_OVERRIDE_PATH}/${OPTARG}.${FILE_SUFFIX}")
        LOCAL_TEMPLATE_PATH="${TEMPLATE_OVERRIDE_PATH}/${OPTARG}.${FILE_SUFFIX}"
      else
        if [[ ! -f "templates/${OPTARG}.${DEFAULT_SUFFIX}" ]]; then
            if [[ -f "templates/${OPTARG}.${ALTERNATIVE_SUFFIX}" ]]; then
              FILE_SUFFIX=$ALTERNATIVE_SUFFIX
            fi
          fi
        TEMPLATE_FILE=("--template-body" "file://templates/${OPTARG}.${FILE_SUFFIX}")
        LOCAL_TEMPLATE_PATH="templates/${OPTARG}.${FILE_SUFFIX}"
        LOCAL_TEMPLATE_FILENAME="${OPTARG}.${FILE_SUFFIX}"
      fi
      ;;
    t)
      TAG_FILE="tags/${OPTARG}.json"
      RAW_TAG_FILE="${OPTARG}"
      ;;
    e)
      EXTRA_TAGFILE=$OPTARG
      ;;
    m)
      MANUAL_TAGS=$OPTARG
      ;;
    p)
      PARAMETER_FILE="parameters/${OPTARG}.json"
      RAW_PARAMETER_FILE="${OPTARG}"
      ;;
    c)
      COMMON_PARAMETER_FILE="${OPTARG}"
      ;;
    o)
      OVERRIDE_PARAMS=$OPTARG
      ;;
    s)
      ARTEFACT_BUCKET="${OPTARG}"
      ;;
    q)
      CREATE_CHANGESET="${OPTARG}"
      CHANGESETNAME="${OPTARG}"
      FOG_FLAG="--dry-run"
      ;;
    y)
      DEPLOY_CHANGESET="${OPTARG}"
      CHANGESETNAME="${OPTARG}"
      ;;
    x)
      DELETE_CHANGESET="${OPTARG}"
      CHANGESETNAME="${OPTARG}"
      ;;
    a)
      DEPLOY_CREATE_CHANGESET="${OPTARG}"
      CHANGESETNAME="${OPTARG}"
      FOG_FLAG="--non-interactive"
      ;;
    h)
      full_help
      exit 2
      ;;
    *)
      echo "Invalid argument supplied, exiting"
      exit 1
  esac
done



# Semver version comparison for fog version check
vercomp () {
    if [[ "$1" == "$2" ]]
    then
        return 0
    fi
    local IFS=.
    local i ver1=("$1") ver2=("$2")
    # fill empty fields in ver1 with zeros
    for ((i=${#ver1[@]}; i<${#ver2[@]}; i++))
    do
        ver1[i]=0
    done
    for ((i=0; i<${#ver1[@]}; i++))
    do
        if [[ -z ${ver2[i]} ]]
        then
            # fill empty fields in ver2 with zeros
            ver2[i]=0
        fi
        if ((10#${ver1[i]} > 10#${ver2[i]}))
        then
            return 1
        fi
        if ((10#${ver1[i]} < 10#${ver2[i]}))
        then
            return 2
        fi
    done
    return 0
}

usefog () {
  if [[ -x "$(command -v fog)" ]]; then
    FOG_VERSION=$(fog version)
    if [[ "$FOG_VERSION" == "dev" ]] && [[ "$ALLOW_DEV_FOG" != true ]]; then
      echo "Found dev version of fog, which is not allowed, using shell script"
      return
    else
      vercomp "$FOG_VERSION" "$MINIMUM_FOG_VERSION"
      case $? in
        0) op='=';;
        1) op='>';;
        2) op='<';;
      esac
      if [[ $op == "<" ]]; then
        echo "Unfortunately your version of fog is too old for this script. Please upgrade to at least version ${MINIMUM_FOG_VERSION}"
        return
      fi
      echo "Fog version '${FOG_VERSION}' found, using fog"
      fog deploy --stackname "${NAME}" --file "${RAW_TEMPLATE_FILE}" --parameters "${RAW_PARAMETER_FILE}" --tags "${RAW_TAG_FILE}" "${FOG_FLAG}" --changeset "${CHANGESETNAME}"
      exit $?
    fi
  fi
}

usefog

REGION=$(aws configure list | grep region | awk '{print $2}')

function checkDeployment {
    if [[ "${CHANGESET_TYPE}" == "UPDATE" ]]; then
      SUCCESS_TEXT="Finished updating ${NAME}"
      aws cloudformation wait stack-update-complete --stack-name "${NAME}"
      RESULT=$?
    elif [[ "${CHANGESET_TYPE}" == "CREATE" ]]; then
      SUCCESS_TEXT="Finished creating ${NAME}"
      aws cloudformation wait stack-create-complete --stack-name "${NAME}"
      RESULT=$?
    fi

    if [[ ${RESULT} == 0 ]]; then
      echo "${SUCCESS_TEXT}"
      echo "Outputs from the stack:"
      aws cloudformation describe-stacks --stack-name "${NAME}" --query 'Stacks[*].[Outputs]' --output table
      exit ${RESULT}
    else
      # There are cases where a deployment takes over an hour.
      # This does an additional status check when that's the case so it will try again
      status=$(aws cloudformation describe-stacks --stack-name "${NAME}" --query "Stacks[0].StackStatus" --output text)
      if [[ "$status" == "CREATE_IN_PROGRESS" ]] || [[ "$status" == "UPDATE_IN_PROGRESS" ]]; then
       checkDeployment
      fi
      echo "Stack ${CHANGESET_TYPE} failed. Below are the latest failures (please keep in mind some may be from earlier updates)."
      aws cloudformation describe-stack-events --stack-name "${NAME}" --query "StackEvents[?contains(ResourceStatus, 'FAILED')].{CfnId:LogicalResourceId,Status:ResourceStatus,Reason:ResourceStatusReason,Timestamp:Timestamp}" --output table
      echo "See the following link for a full overview of all events: https://console.aws.amazon.com/cloudformation/home?region=${REGION}#/stacks/events?stackId=${stackarn}"
      if [[ ("${CHANGESET_TYPE}" == "CREATE") && (-z $DEPLOY_CHANGESET) ]]; then
        if [[ -n $DEPLOY_CREATE_CHANGESET ]]; then
          echo "This stack was new. Automatically deleting the stack, you can still look up the errors at the above link.";
          aws cloudformation delete-stack --stack-name "${NAME}"
          echo "Deletion completed, you can now try again";
          exit ${RESULT}
        fi
        while true; do
          read -r -p "Because this is a new stack, you have to delete it before you can rebuild it. Do you wish to do so now? " yn
          case $yn in
              [Yy]* ) echo "Deleting the stack now"; aws cloudformation delete-stack --stack-name "${NAME}"; echo "Deletion completed, you can now try again"; break;;
              [Nn]* ) echo "Ok, leaving the stack intact"; break;;
              * ) echo "Please answer yes or no.";;
          esac
        done
      fi
      exit ${RESULT}
    fi
}

function deployChangeSet {
  echo "Applying change set ${CHANGESETNAME}"
  CHANGES_COUNT=$(aws cloudformation list-change-sets --stack-name "${NAME}" --output json | jq ".Summaries | map(select(.ChangeSetName == \"${CHANGESETNAME}\")) | length")
  RESULT=$?
  if [[ ${RESULT} != 0 ]]; then
    echo "Failed to lookup change set [${CHANGESETNAME}] on stack ${NAME}"
    exit ${RESULT}
  fi
  stackarn=$(aws cloudformation describe-change-set --change-set-name "${CHANGESETNAME}" --stack-name "${NAME}"  --query 'StackId' --output text)

  if [ "${CHANGES_COUNT}" == "1" ]; then
    aws cloudformation execute-change-set --change-set-name "${CHANGESETNAME}" --stack-name "${NAME}"
    RESULT=$?
    if [[ ${RESULT} != 0 ]]; then
      echo "Failed to execute change set [${CHANGESETNAME}] on stack ${NAME}"
      exit ${RESULT}
    fi
    checkDeployment
  else
    echo "Change set with name [${CHANGESETNAME}] does not exist"
    exit 0
  fi
}

# deleteChangeset will delete a changeset and if it's a new stack, will delete
# the stack as well so it won't be stuck in REVIEW_IN_PROGRESS
function deleteChangeset {
  aws cloudformation delete-change-set --change-set-name "${CHANGESETNAME}" --stack-name "${NAME}"
  stackstatus=$(aws cloudformation describe-stacks --stack-name "${NAME}" --query 'Stacks[0].StackStatus' --output text)
  if [[ "${stackstatus}" == "REVIEW_IN_PROGRESS" ]]; then
    echo "Stack is in REVIEW_IN_PROGRESS"
    resourcecount=$(aws cloudformation describe-stack-resources --stack-name "${NAME}" --query 'StackResources[*]| length(@)')
    if [[ "${resourcecount}" == "0" ]]; then
      echo "Stack has no resources. Deleting stack"
      aws cloudformation delete-stack --stack-name "${NAME}"
    fi
  fi
}


if [[ -n $EXTRA_TAGFILE ]]; then
  tmptags=$(jq -s '. | flatten' "${TAG_FILE}" "${EXTRA_TAGFILE}")
  tmpfile=$(mktemp /tmp/tagfile.XXXXXX)
  echo "${tmptags}" >> "${tmpfile}"
  TAG_FILE=${tmpfile}
fi

if [[ -n $MANUAL_TAGS ]]; then
  tmpmanualtagsfile=$(mktemp /tmp/manualtagfile.XXXXXX)
  echo "${MANUAL_TAGS}" >> "${tmpmanualtagsfile}"
  tmptags=$(jq -s '. | flatten' "${TAG_FILE}" "${tmpmanualtagsfile}")
  tmpfile=$(mktemp /tmp/tagfile.XXXXXX)
  echo "${tmptags}" >> "${tmpfile}"
  TAG_FILE=${tmpfile}
  rm "${tmpmanualtagsfile}"
fi

if [[ -n $COMMON_PARAMETER_FILE ]]; then
  tmpparams=$(jq -s '.[0] + .[1] | unique_by(.ParameterKey)' "${PARAMETER_FILE}" "${COMMON_PARAMETER_FILE}")
  tmpfile=$(mktemp /tmp/paramfile.XXXXXX)
  echo "${tmpparams}" >> "${tmpfile}"
  PARAMETER_FILE=${tmpfile}
fi

if [[ -n $OVERRIDE_PARAMS ]]; then
  tmpoverrideparamsfile=$(mktemp /tmp/overrideparamsfile.XXXXXX)
  echo "${OVERRIDE_PARAMS}" >> "${tmpoverrideparamsfile}"
  overrideparams=$(jq -s '. | flatten' "${PARAMETER_FILE}" "${tmpoverrideparamsfile}")
  tmpfile=$(mktemp /tmp/paramfile.XXXXXX)
  echo "${overrideparams}" >> "${tmpfile}"
  PARAMETER_FILE=${tmpfile}
  rm "${tmpoverrideparamsfile}"
fi

PARAMETER_FILE_STRING=""
if [[ -n $PARAMETER_FILE ]]; then
  PARAMETER_FILE_STRING=("--parameters" "file://${PARAMETER_FILE}")
fi

# Generate a changesetname if it doesn't exist yet
if [[ -z $CHANGESETNAME ]]; then
  CHANGESETNAME=$(date "+A%Y-%m-%d-%H-%M-%S")
fi
if [[ -n $DEPLOY_CHANGESET ]]; then
  CHANGESET_TYPE="UPDATE"
  STACK_REVIEW_IN_PROGRESS=$(aws cloudformation list-stacks --stack-status-filter REVIEW_IN_PROGRESS --query "StackSummaries[?StackName=='${NAME}'].StackName" --output text)
  if [[ "$STACK_REVIEW_IN_PROGRESS" == "$NAME" ]]; then
    CHANGESET_TYPE="CREATE"
  fi
  deployChangeSet
  exit
fi
if [[ -n $DELETE_CHANGESET ]]; then
  deleteChangeset
  exit
fi

if [[ -n $ARTEFACT_BUCKET ]]; then
  ARTEFACT_LOCATION="s3://${ARTEFACT_BUCKET}/${CHANGESETNAME}-${LOCAL_TEMPLATE_FILENAME}"
  # TODO see if there's a clean way to get the https URL of an object
  TEMPLATE_FILE=("--template-url" "https://${ARTEFACT_BUCKET}.s3-${REGION}.amazonaws.com/${CHANGESETNAME}-${LOCAL_TEMPLATE_FILENAME}")
  echo "Uploading template to S3 bucket ${ARTEFACT_LOCATION}"
  aws s3 cp "${LOCAL_TEMPLATE_PATH}" "${ARTEFACT_LOCATION}"
fi

CHANGESET_TYPE="UPDATE"
if ! aws cloudformation describe-stacks --stack-name "${NAME}" >/dev/null 2>&1 ; then
  CHANGESET_TYPE="CREATE"
fi

echo "Creating a change set for ${NAME}"
if ! aws cloudformation create-change-set "${TEMPLATE_FILE[@]}" --change-set-type "${CHANGESET_TYPE}" --stack-name "${NAME}" --change-set-name "${CHANGESETNAME}" "${PARAMETER_FILE_STRING[@]}" --tags "file://${TAG_FILE}" --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND >/dev/null; then
  echo "There was a failure creating the changeset, please review the above error message"
  exit 1
fi
# Add a 10 second sleep to prevent the 30 second delay from the wait function
sleep 10
aws cloudformation wait change-set-create-complete --change-set-name "${CHANGESETNAME}" --stack-name "${NAME}" >/dev/null 2>&1
status=$(aws cloudformation describe-change-set --change-set-name "${CHANGESETNAME}" --stack-name "${NAME}" --query Status --output text)
statusreason=$(aws cloudformation describe-change-set --change-set-name "${CHANGESETNAME}" --stack-name "${NAME}" --query StatusReason --output text)
# Check if there are no updates in which case, clean up and move on
# Below statusreason value also returns status=FAILED, this needs to be evaluated first
if [[ "$statusreason" == "No updates are to be performed." || "$statusreason" == "The submitted information didn't contain changes. Submit different information to create a change set." ]]; then
    echo "No updates are to be performed on ${NAME}"
    aws cloudformation delete-change-set --change-set-name "${CHANGESETNAME}" --stack-name "${NAME}" >/dev/null 2>&1
    exit 0;
fi
if [[ "$status" == "FAILED" ]]; then
  echo "Changeset failed with reason: ${statusreason}"
  exit 1
fi
# Add the changeset
echo "Updates have been found! Please review before continuing."
aws cloudformation describe-change-set --change-set-name "${CHANGESETNAME}" --stack-name "${NAME}" --query 'Changes[*].ResourceChange.{Action:Action,CfnName:LogicalResourceId,Type:ResourceType,ID:PhysicalResourceId,Replacement:Replacement}' --output table
stackarn=$(aws cloudformation describe-change-set --change-set-name "${CHANGESETNAME}" --stack-name "${NAME}"  --query 'StackId' --output text)
changesetarn=$(aws cloudformation describe-change-set --change-set-name "${CHANGESETNAME}" --stack-name "${NAME}"  --query 'ChangeSetId' --output text)
echo "You can view the details in the Console: https://ap-southeast-2.console.aws.amazon.com/cloudformation/home?region=${REGION}#/stacks/changesets/changes?stackId=${stackarn}&changeSetId=${changesetarn}"
echo "The stack name is ${NAME}"
echo "The changeset name is ${CHANGESETNAME}"
if [[ -n $CREATE_CHANGESET ]]; then
  echo "The changeset was only meant to be created, not deployed"
  exit 0
fi
if [[ -n $DEPLOY_CREATE_CHANGESET ]]; then
  echo "Deploying automatically"
  deployChangeSet
fi
while true; do
    read -r -p "Do you wish to apply this changeset? " yn
    case $yn in
        [Yy]* ) break;;
        [Nn]* ) echo "Deleting changeset"; deleteChangeset; exit;;
        * ) echo "Please answer yes or no.";;
    esac
done

deployChangeSet