CREATE TABLE Orgs (
  OrgLogin STRING(MAX) NOT NULL,
  Company STRING(MAX),
  Description STRING(MAX),
  AvatarURL STRING(MAX),
) PRIMARY KEY(OrgLogin);

CREATE TABLE Maintainers (
  OrgLogin STRING(MAX) NOT NULL,
  UserLogin STRING(MAX) NOT NULL,
  Paths ARRAY<STRING(MAX)>,
  Emeritus BOOL NOT NULL,
  CachedInfo STRING(MAX) NOT NULL,
) PRIMARY KEY(OrgLogin, UserLogin),
  INTERLEAVE IN PARENT Orgs ON DELETE CASCADE;

CREATE TABLE Members (
  OrgLogin STRING(MAX) NOT NULL,
  UserLogin STRING(MAX) NOT NULL,
  CachedInfo STRING(MAX),
) PRIMARY KEY(OrgLogin, UserLogin),
  INTERLEAVE IN PARENT Orgs ON DELETE CASCADE;

CREATE TABLE Repos (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  Description STRING(MAX) NOT NULL,
  RepoNumber INT64 NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName),
  INTERLEAVE IN PARENT Orgs ON DELETE CASCADE;

CREATE TABLE BotActivity (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  LastIssueSyncStart TIMESTAMP NOT NULL,
  LastIssueCommentSyncStart TIMESTAMP NOT NULL,
  LastPullRequestReviewCommentSyncStart TIMESTAMP NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE CoverageData (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  BranchName STRING(MAX) NOT NULL,
  PackageName STRING(MAX) NOT NULL,
  Sha STRING(MAX) NOT NULL,
  TestName STRING(MAX) NOT NULL,
  Type STRING(MAX) NOT NULL,
  CompletedAt TIMESTAMP NOT NULL,
  StmtsCovered INT64 NOT NULL,
  StmtsTotal INT64 NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, BranchName, PackageName, Sha, TestName),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE IssueCommentEvents (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  IssueNumber INT64 NOT NULL,
  IssueCommentID INT64 NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
  Action STRING(MAX) NOT NULL,
  Actor STRING(MAX) NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, IssueNumber, IssueCommentID, CreatedAt),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE IssueComments (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  IssueNumber INT64 NOT NULL,
  IssueCommentID INT64 NOT NULL,
  Author STRING(MAX) NOT NULL,
  Body STRING(MAX) NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
  UpdatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, IssueNumber, IssueCommentID),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE IssueEvents (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
  IssueNumber INT64 NOT NULL,
  Actor STRING(MAX) NOT NULL,
  Action STRING(MAX) NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, CreatedAt),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE IssuePipelines (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  IssueNumber INT64 NOT NULL,
  Pipeline STRING(MAX) NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, IssueNumber),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE Issues (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  IssueNumber INT64 NOT NULL,
  Title STRING(MAX) NOT NULL,
  Body STRING(MAX) NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
  UpdatedAt TIMESTAMP NOT NULL,
  ClosedAt TIMESTAMP NOT NULL,
  State STRING(MAX) NOT NULL,
  Author STRING(MAX) NOT NULL,
  Assignees ARRAY<STRING(MAX)>,
  Labels ARRAY<STRING(MAX)>,
) PRIMARY KEY(OrgLogin, RepoName, IssueNumber),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE Labels (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  LabelName STRING(MAX) NOT NULL,
  Description STRING(MAX) NOT NULL,
  Color STRING(MAX) NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, LabelName),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE PullRequestEvents (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  PullRequestNumber INT64 NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
  Action STRING(MAX) NOT NULL,
  Actor STRING(MAX) NOT NULL,
  Merged BOOL NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, PullRequestNumber, CreatedAt),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE PullRequestReviewCommentEvents (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  PullRequestNumber INT64 NOT NULL,
  PullRequestReviewCommentID INT64 NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
  Action STRING(MAX) NOT NULL,
  Actor STRING(MAX) NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, PullRequestNumber, PullRequestReviewCommentID, CreatedAt),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE PullRequestReviewComments (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  PullRequestNumber INT64 NOT NULL,
  PullRequestReviewCommentID INT64 NOT NULL,
  Author STRING(MAX) NOT NULL,
  Body STRING(MAX) NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
  UpdatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, PullRequestNumber, PullRequestReviewCommentID),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE PullRequestReviewEvents (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  PullRequestNumber INT64 NOT NULL,
  PullRequestReviewID INT64 NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
  Action STRING(MAX) NOT NULL,
  Actor STRING(MAX) NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, PullRequestNumber, PullRequestReviewID, CreatedAt),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE PullRequestReviews (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  PullRequestNumber INT64 NOT NULL,
  PullRequestReviewID INT64 NOT NULL,
  Author STRING(MAX) NOT NULL,
  Body STRING(MAX) NOT NULL,
  SubmittedAt TIMESTAMP NOT NULL,
  State STRING(MAX) NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, PullRequestNumber, PullRequestReviewID),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE PullRequests (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  PullRequestNumber INT64 NOT NULL,
  UpdatedAt TIMESTAMP NOT NULL,
  RequestedReviewers ARRAY<STRING(MAX)>,
  Files ARRAY<STRING(MAX)>,
  State STRING(MAX) NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
  ClosedAt TIMESTAMP NOT NULL,
  MergedAt TIMESTAMP NOT NULL,
  Labels ARRAY<STRING(MAX)>,
  Author STRING(MAX) NOT NULL,
  Assignees ARRAY<STRING(MAX)>,
  Title STRING(MAX) NOT NULL,
  Body STRING(MAX) NOT NULL,
  HeadCommit STRING(MAX) NOT NULL,
  BranchName STRING(MAX) NOT NULL,
  Merged BOOL NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, PullRequestNumber),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE INDEX AuthorIndex ON PullRequests(Author);

CREATE TABLE RepoCommentEvents (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  RepoCommentID INT64 NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
  Action STRING(MAX) NOT NULL,
  Actor STRING(MAX) NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, RepoCommentID, CreatedAt),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE RepoComments (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  CommentID INT64 NOT NULL,
  Body STRING(MAX) NOT NULL,
  Author STRING(MAX) NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
  UpdatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, CommentID),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE MonitorStatus (
  MonitorName STRING(MAX) NOT NULL,
  Status STRING(MAX) NOT NULL,
  ProjectID STRING(MAX),
  TestID STRING(MAX) NOT NULL,
  Description STRING(MAX),
  UpdatedTime TIMESTAMP NOT NULL,
  FiredTimes INT64 NOT NULL,
  LastFiredTime TIMESTAMP,
  IsActive BOOL,
) PRIMARY KEY(TestID, MonitorName)

CREATE TABLE ReleaseQualTestMetadata (
  ClusterName STRING(MAX) NOT NULL,
  ProjectID STRING(MAX) NOT NULL,
  TestID STRING(MAX) NOT NULL,
  Branch STRING(MAX) NOT NULL,
  PrometheusLink STRING(MAX),
  GrafanaLink STRING(MAX),
) PRIMARY KEY(TestID)

CREATE TABLE TestResults (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  PullRequestNumber INT64 NOT NULL,
  RunNumber INT64 NOT NULL,
  TestName STRING(MAX) NOT NULL,
  Done BOOL NOT NULL,
  BaseSha STRING(MAX),
  CloneFailed BOOL NOT NULL,
  FinishTime TIMESTAMP,
  Result STRING(MAX),
  RunPath STRING(MAX) NOT NULL,
  Sha BYTES(MAX) NOT NULL,
  StartTime TIMESTAMP NOT NULL,
  TestPassed BOOL,
  HasArtifacts BOOL,
  Signatures ARRAY<STRING(MAX)>,
  Artifacts ARRAY<STRING(MAX)>,
) PRIMARY KEY(OrgLogin, RepoName, TestName, PullRequestNumber, RunNumber, Done),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE PostSubmitTestResults (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  RunNumber INT64 NOT NULL,
  TestName STRING(MAX) NOT NULL,
  Done BOOL NOT NULL,
  BaseSha STRING(MAX),
  CloneFailed BOOL NOT NULL,
  FinishTime TIMESTAMP,
  Result STRING(MAX),
  RunPath STRING(MAX) NOT NULL,
  Sha BYTES(MAX) NOT NULL,
  StartTime TIMESTAMP NOT NULL,
  TestPassed BOOL,
  HasArtifacts BOOL,
  Signatures ARRAY<STRING(MAX)>,
  Artifacts ARRAY<STRING(MAX)>,
) PRIMARY KEY(OrgLogin, RepoName, TestName, BaseSha, RunNumber, Done),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE SuiteOutcomes (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  RunNumber INT64 NOT NULL,
  TestName STRING(MAX) NOT NULL,
  BaseSha STRING(MAX),
  Done BOOL NOT NULL,
  SuiteName STRING(MAX) NOT NULL,
  Environment STRING(MAX) NOT NULL,
  Multicluster BOOL NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, TestName, BaseSha, RunNumber, Done, SuiteName),
  INTERLEAVE IN PARENT PostSubmitTestResults ON DELETE CASCADE;

CREATE TABLE TestOutcomes (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  RunNumber INT64 NOT NULL,
  TestName STRING(MAX) NOT NULL,
  BaseSha STRING(MAX),
  Done BOOL NOT NULL,
  SuiteName STRING(MAX) NOT NULL,
  TestOutcomeName STRING(MAX) NOT NULL,
  Type STRING(MAX) NOT NULL,
  Outcome STRING(MAX) NOT NULL,
) PRIMARY KEY(OrgLogin, RepoName, TestName, BaseSha, RunNumber, Done, SuiteName, TestOutcomeName),
  INTERLEAVE IN PARENT SuiteOutcomes ON DELETE CASCADE;

CREATE TABLE FeatureLabels (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  RunNumber INT64 NOT NULL,
  TestName STRING(MAX) NOT NULL,
  BaseSha STRING(MAX),
  Done BOOL NOT NULL,
  SuiteName STRING(MAX) NOT NULL,
  TestOutcomeName STRING(MAX) NOT NULL,
  Label STRING(MAX) NOT NULL,
  Scenario ARRAY<STRING(MAX)>,
) PRIMARY KEY(OrgLogin, RepoName, TestName, BaseSha, RunNumber, Done, SuiteName, TestOutcomeName),
  INTERLEAVE IN PARENT TestOutcomes ON DELETE CASCADE;

CREATE TABLE ConfirmedFlakes (
  OrgLogin STRING(MAX) NOT NULL,
  RepoName STRING(MAX) NOT NULL,
  PullRequestNumber INT64 NOT NULL,
  RunNumber INT64 NOT NULL,
  TestName STRING(MAX) NOT NULL,
  Done BOOL NOT NULL,
  PassingRunNumber INT64 NOT NULL,
  IssueNum INT64,
) PRIMARY KEY(OrgLogin, RepoName, TestName, PullRequestNumber, RunNumber, Done, PassingRunNumber),
  INTERLEAVE IN PARENT TestResults ON DELETE NO ACTION;

CREATE TABLE Users (
  UserLogin STRING(MAX) NOT NULL,
  Name STRING(MAX) NOT NULL,
  Company STRING(MAX) NOT NULL,
  AvatarUrl STRING(MAX) NOT NULL,
) PRIMARY KEY(UserLogin);

CREATE TABLE UserAffiliation (
  UserLogin STRING(MAX) NOT NULL,
  StartTime TIMESTAMP NOT NULL,
  EndTime TIMESTAMP NOT NULL,
  Organization STRING(MAX) NOT NULL,
  Counter INT64,
) PRIMARY KEY(UserLogin, Counter),
  INTERLEAVE IN PARENT Users ON DELETE CASCADE
