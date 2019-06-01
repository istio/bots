CREATE TABLE BotActivity (
  LastSyncStart TIMESTAMP,
  LastSyncEnd TIMESTAMP,
) PRIMARY KEY(LastSyncStart);

CREATE TABLE Orgs (
  OrgID STRING(MAX) NOT NULL,
  Login STRING(MAX) NOT NULL,
) PRIMARY KEY(OrgID);

CREATE UNIQUE INDEX OrgsLogin ON Orgs(Login);

CREATE TABLE Maintainers (
  OrgID STRING(MAX) NOT NULL,
  UserID STRING(MAX) NOT NULL,
  Paths ARRAY<STRING(MAX)>,
) PRIMARY KEY(OrgID, UserID),
  INTERLEAVE IN PARENT Orgs ON DELETE CASCADE;

CREATE TABLE Members (
  OrgID STRING(MAX) NOT NULL,
  UserID STRING(MAX) NOT NULL,
) PRIMARY KEY(OrgID, UserID),
  INTERLEAVE IN PARENT Orgs ON DELETE CASCADE;

CREATE TABLE Repos (
  OrgID STRING(MAX) NOT NULL,
  RepoID STRING(MAX) NOT NULL,
  Name STRING(MAX) NOT NULL,
  Description STRING(MAX),
) PRIMARY KEY(OrgID, RepoID),
  INTERLEAVE IN PARENT Orgs ON DELETE CASCADE;

CREATE UNIQUE INDEX ReposName ON Repos(OrgID, Name);

CREATE TABLE Issues (
  OrgID STRING(MAX) NOT NULL,
  RepoID STRING(MAX) NOT NULL,
  IssueID STRING(MAX) NOT NULL,
  Number INT64 NOT NULL,
  Title STRING(MAX),
  Body STRING(MAX),
  CreatedAt TIMESTAMP NOT NULL,
  UpdatedAt TIMESTAMP,
  ClosedAt TIMESTAMP,
  State STRING(MAX),
  AuthorID STRING(MAX) NOT NULL,
  AssigneeIDs ARRAY<STRING(MAX)>,
  LabelIDs ARRAY<STRING(MAX)>,
) PRIMARY KEY(OrgID, RepoID, IssueID),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE UNIQUE INDEX IssuesNumber ON Issues(RepoID, Number);

CREATE TABLE IssueComments (
  OrgID STRING(MAX) NOT NULL,
  RepoID STRING(MAX) NOT NULL,
  IssueID STRING(MAX) NOT NULL,
  IssueCommentID STRING(MAX) NOT NULL,
  AuthorID STRING(MAX) NOT NULL,
  Body STRING(MAX),
  CreatedAt TIMESTAMP NOT NULL,
  UpdatedAt TIMESTAMP,
) PRIMARY KEY(OrgID, RepoID, IssueID, IssueCommentID),
  INTERLEAVE IN PARENT Issues ON DELETE CASCADE;

CREATE TABLE Labels (
  OrgID STRING(MAX) NOT NULL,
  RepoID STRING(MAX) NOT NULL,
  LabelID STRING(MAX) NOT NULL,
  Name STRING(MAX) NOT NULL,
  Description STRING(MAX),
) PRIMARY KEY(OrgID, RepoID, LabelID),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE PullRequests (
  OrgID STRING(MAX) NOT NULL,
  RepoID STRING(MAX) NOT NULL,
  IssueID STRING(MAX) NOT NULL,
  UpdatedAt TIMESTAMP,
  RequestedReviewerIDs ARRAY<STRING(MAX)>,
  Files ARRAY<STRING(MAX)>,
) PRIMARY KEY(OrgID, RepoID, IssueID),
  INTERLEAVE IN PARENT Repos ON DELETE CASCADE;

CREATE TABLE PullRequestReviews (
  OrgID STRING(MAX) NOT NULL,
  RepoID STRING(MAX) NOT NULL,
  IssueID STRING(MAX) NOT NULL,
  PullRequestReviewID STRING(MAX) NOT NULL,
  AuthorID STRING(MAX) NOT NULL,
  Body STRING(MAX),
  SubmittedAt TIMESTAMP NOT NULL,
  State STRING(MAX) NOT NULL,
) PRIMARY KEY(OrgID, RepoID, IssueID, PullRequestReviewID),
  INTERLEAVE IN PARENT PullRequests ON DELETE CASCADE;

CREATE TABLE Users (
  UserID STRING(MAX) NOT NULL,
  Login STRING(MAX),
  Name STRING(MAX),
  Company STRING(MAX),
) PRIMARY KEY(UserID);

CREATE UNIQUE INDEX UsersLogin ON Users(Login)
