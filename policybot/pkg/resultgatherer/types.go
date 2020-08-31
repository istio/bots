package resultgatherer

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)
// ProwJobType specifies how the job is triggered.
type ProwJobType string

// Various job types.
const (
	// PresubmitJob means it runs on unmerged PRs.
	PresubmitJob ProwJobType = "presubmit"
	// PostsubmitJob means it runs on each new commit.
	PostsubmitJob ProwJobType = "postsubmit"
	// Periodic job means it runs on a time-basis, unrelated to git changes.
	PeriodicJob ProwJobType = "periodic"
	// BatchJob tests multiple unmerged PRs at the same time.
	BatchJob ProwJobType = "batch"
)

// ProwJobState specifies whether the job is running
type ProwJobState string

// Various job states.
const (
	// TriggeredState means the job has been created but not yet scheduled.
	TriggeredState ProwJobState = "triggered"
	// PendingState means the job is currently running and we are waiting for it to finish.
	PendingState ProwJobState = "pending"
	// SuccessState means the job completed without error (exit 0)
	SuccessState ProwJobState = "success"
	// FailureState means the job completed with errors (exit non-zero)
	FailureState ProwJobState = "failure"
	// AbortedState means prow killed the job early (new commit pushed, perhaps).
	AbortedState ProwJobState = "aborted"
	// ErrorState means the job could not schedule (bad config, perhaps).
	ErrorState ProwJobState = "error"
)

// ProwJobAgent specifies the controller (such as plank or jenkins-agent) that runs the job.
type ProwJobAgent string

const (
	// KubernetesAgent means prow will create a pod to run this job.
	KubernetesAgent ProwJobAgent = "kubernetes"
	// JenkinsAgent means prow will schedule the job on jenkins.
	JenkinsAgent ProwJobAgent = "jenkins"
	// TektonAgent means prow will schedule the job via a tekton PipelineRun CRD resource.
	TektonAgent = "tekton-pipeline"
)

const (
	// DefaultClusterAlias specifies the default cluster key to schedule jobs.
	DefaultClusterAlias = "default"
)

const (
	// StartedStatusFile is the JSON file that stores information about the build
	// at the start ob the build. See testgrid/metadata/job.go for more details.
	StartedStatusFile = "started.json"

	// FinishedStatusFile is the JSON file that stores information about the build
	// after its completion. See testgrid/metadata/job.go for more details.
	FinishedStatusFile = "finished.json"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProwJob contains the spec as well as runtime metadata.
type ProwJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProwJobSpec   `json:"spec,omitempty"`
	Status ProwJobStatus `json:"status,omitempty"`
}

// ProwJobSpec configures the details of the prow job.
//
// Details include the podspec, code to clone, the cluster it runs
// any child jobs, concurrency limitations, etc.
type ProwJobSpec struct {
	// Type is the type of job and informs how
	// the jobs is triggered
	Type ProwJobType `json:"type,omitempty"`
	// Agent determines which controller fulfills
	// this specific ProwJobSpec and runs the job
	Agent ProwJobAgent `json:"agent,omitempty"`
	// Cluster is which Kubernetes cluster is used
	// to run the job, only applicable for that
	// specific agent
	Cluster string `json:"cluster,omitempty"`
	// Namespace defines where to create pods/resources.
	Namespace string `json:"namespace,omitempty"`
	// Job is the name of the job
	Job string `json:"job,omitempty"`
	// Report determines if the result of this job should
	// be reported (e.g. status on GitHub, message in Slack, etc.)
	Report bool `json:"report,omitempty"`
	// Context is the name of the status context used to
	// report back to GitHub
	Context string `json:"context,omitempty"`
	// RerunCommand is the command a user would write to
	// trigger this job on their pull request
	RerunCommand string `json:"rerun_command,omitempty"`
	// MaxConcurrency restricts the total number of instances
	// of this job that can run in parallel at once
	MaxConcurrency int `json:"max_concurrency,omitempty"`
	// ErrorOnEviction indicates that the ProwJob should be completed and given
	// the ErrorState status if the pod that is executing the job is evicted.
	// If this field is unspecified or false, a new pod will be created to replace
	// the evicted one.
	ErrorOnEviction bool `json:"error_on_eviction,omitempty"`

	// PodSpec provides the basis for running the test under
	// a Kubernetes agent
	PodSpec *corev1.PodSpec `json:"pod_spec,omitempty"`

	// RerunAuthConfig holds information about which users can rerun the job
	RerunAuthConfig *RerunAuthConfig `json:"rerun_auth_config,omitempty"`

	// Hidden specifies if the Job is considered hidden.
	// Hidden jobs are only shown by deck instances that have the
	// `--hiddenOnly=true` or `--show-hidden=true` flag set.
	// Presubmits and Postsubmits can also be set to hidden by
	// adding their repository in Decks `hidden_repo` setting.
	Hidden bool `json:"hidden,omitempty"`
}

type GitHubTeamSlug struct {
	Slug string `json:"slug"`
	Org  string `json:"org"`
}

type RerunAuthConfig struct {
	// If AllowAnyone is set to true, any user can rerun the job
	AllowAnyone bool `json:"allow_anyone,omitempty"`
	// GitHubTeams contains IDs of GitHub teams of users who can rerun the job
	// If you know the name of a team and the org it belongs to,
	// you can look up its ID using this command, where the team slug is the hyphenated name:
	// curl -H "Authorization: token <token>" "https://api.github.com/orgs/<org-name>/teams/<team slug>"
	// or, to list all teams in a given org, use
	// curl -H "Authorization: token <token>" "https://api.github.com/orgs/<org-name>/teams"
	GitHubTeamIDs []int `json:"github_team_ids,omitempty"`
	// GitHubTeamSlugs contains slugs and orgs of teams of users who can rerun the job
	GitHubTeamSlugs []GitHubTeamSlug `json:"github_team_slugs,omitempty"`
	// GitHubUsers contains names of individual users who can rerun the job
	GitHubUsers []string `json:"github_users,omitempty"`
	// GitHubOrgs contains names of GitHub organizations whose members can rerun the job
	GitHubOrgs []string `json:"github_orgs,omitempty"`
}

// ProwJobStatus provides runtime metadata, such as when it finished, whether it is running, etc.
type ProwJobStatus struct {
	// StartTime is equal to the creation time of the ProwJob
	StartTime metav1.Time `json:"startTime,omitempty"`
	// PendingTime is the timestamp for when the job moved from triggered to pending
	PendingTime *metav1.Time `json:"pendingTime,omitempty"`
	// CompletionTime is the timestamp for when the job goes to a final state
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
	State          ProwJobState `json:"state,omitempty"`
	Description    string       `json:"description,omitempty"`
	URL            string       `json:"url,omitempty"`

	// PodName applies only to ProwJobs fulfilled by
	// plank. This field should always be the same as
	// the ProwJob.ObjectMeta.Name field.
	PodName string `json:"pod_name,omitempty"`

	// BuildID is the build identifier vended either by tot
	// or the snowflake library for this job and used as an
	// identifier for grouping artifacts in GCS for views in
	// TestGrid and Gubernator. Idenitifiers vended by tot
	// are monotonically increasing whereas identifiers vended
	// by the snowflake library are not.
	BuildID string `json:"build_id,omitempty"`

	// JenkinsBuildID applies only to ProwJobs fulfilled
	// by the jenkins-operator. This field is the build
	// identifier that Jenkins gave to the build for this
	// ProwJob.
	JenkinsBuildID string `json:"jenkins_build_id,omitempty"`

	// PrevReportStates stores the previous reported prowjob state per reporter
	// So crier won't make duplicated report attempt
	PrevReportStates map[string]ProwJobState `json:"prev_report_states,omitempty"`
}