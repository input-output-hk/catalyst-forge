// permissions.ts
import { Namespace, Context } from "@ory/keto-namespace-types"

// Subjects
export class User implements Namespace { }
export class Client implements Namespace { }

// Root (singleton "forge")
export class Global implements Namespace {
  related: {
    admins: (User | Client)[]
    members: (User | Client)[]
  }

  permits = {
    // Global administrators (root-equivalent)
    admin: (ctx: Context): boolean =>
      this.related.admins.includes(ctx.subject),

    // "Can read platform-wide things" (used to open cross-team read)
    read_all: (ctx: Context): boolean =>
      this.related.members.includes(ctx.subject) || this.permits.admin(ctx),
  }
}

// Teams (frontend, backend, qa, sre, ...)
export class Team implements Namespace {
  related: {
    parent: Global[]                // always [forge]
    admins: (User | Client)[]       // team admins
    maintainers: (User | Client)[]  // can create/manage projects in this team
    members: (User | Client)[]      // normal team members
    viewers: (User | Client)[]      // optional: explicit viewer role
  }

  permits = {
    // Team-level admin tasks (manage team membership, etc.)
    administer: (ctx: Context): boolean =>
      this.related.admins.includes(ctx.subject) ||
      this.related.parent.traverse((g) => g.permits.admin(ctx)),

    // Read: team members, explicit viewers, or any global member (org-wide visibility)
    read: (ctx: Context): boolean =>
      this.related.members.includes(ctx.subject) ||
      this.related.viewers.includes(ctx.subject) ||
      this.related.parent.traverse((g) => g.permits.read_all(ctx)),

    // Create/manage projects in this team
    maintain: (ctx: Context): boolean =>
      this.related.maintainers.includes(ctx.subject) ||
      this.permits.administer(ctx),
  }
}

// Projects belong to a team
export class Project implements Namespace {
  related: {
    team: Team[]
    admins: (User | Client)[]
    maintainers: (User | Client)[]
    builders: (User | Client)[]
    deployers: (User | Client)[]
    viewers: (User | Client)[]
    readers: (User | Client)[]
  }

  permits = {
    // Full admin over the project
    administer: (ctx: Context): boolean =>
      this.related.admins.includes(ctx.subject) ||
      this.related.team.traverse((t) => t.permits.administer(ctx)),


    // Write / manage releases (but not necessarily deploy)
    maintain: (ctx: Context): boolean =>
      this.related.maintainers.includes(ctx.subject) ||
      this.permits.administer(ctx),

    // Build: builders, or anyone who can maintain the project
    build: (ctx: Context): boolean =>
      this.related.builders.includes(ctx.subject) || this.permits.maintain(ctx),

    // Can initiate deployment (project side - env check happens at Deployment)
    deploy: (ctx: Context): boolean =>
      this.related.deployers.includes(ctx.subject) ||
      this.permits.maintain(ctx),

    // Read: readers, or anyone who can read the team, or any global member (cross-team visibility)
    read: (ctx: Context): boolean =>
      this.related.readers.includes(ctx.subject) ||
      this.related.team.traverse((t) => t.permits.read(ctx)),
  }
}

// Releases belong to a project
export class Release implements Namespace {
  related: {
    project: Project[]
    owners: (User | Client)[]
    maintainers: (User | Client)[]
    deployers: (User | Client)[]
    readers: (User | Client)[]
  }

  permits = {
    administer: (ctx: Context): boolean =>
      this.related.owners.includes(ctx.subject) ||
      this.related.project.traverse((p) => p.permits.administer(ctx)),

    maintain: (ctx: Context): boolean =>
      this.related.maintainers.includes(ctx.subject) ||
      this.permits.administer(ctx),

    deploy: (ctx: Context): boolean =>
      this.related.deployers.includes(ctx.subject) ||
      this.permits.maintain(ctx) ||
      this.related.project.traverse((p) => p.permits.deploy(ctx)),

    read: (ctx: Context): boolean =>
      this.related.readers.includes(ctx.subject) ||
      this.permits.maintain(ctx) ||
      this.related.project.traverse((p) => p.permits.read(ctx)),
  }
}

// Environments are shared infra (dev, preprod, prod)
export class Environment implements Namespace {
  related: {
    parent: Global[]                 // always [forge]
    admins: (User | Client)[]
    deployers: (User | Client)[]
    readers: (User | Client)[]
  }

  permits = {
    administer: (ctx: Context): boolean =>
      this.related.admins.includes(ctx.subject) ||
      this.related.parent.traverse((g) => g.permits.admin(ctx)),

    deploy: (ctx: Context): boolean =>
      this.related.deployers.includes(ctx.subject) || this.permits.administer(ctx),

    read: (ctx: Context): boolean =>
      this.related.readers.includes(ctx.subject) ||
      this.related.parent.traverse((g) => g.permits.read_all(ctx)),
  }
}

// Deployment references BOTH a release and an environment
export class Deployment implements Namespace {
  related: {
    release: Release[]
    environment: Environment[]
    readers: (User | Client)[]      // optional: explicit readers
  }

  permits = {
    // operate == "allowed to trigger / approve / rollback a deploy"
    operate: (ctx: Context): boolean =>
      // need BOTH: release permission to deploy AND environment permission to deploy
      (this.related.release.traverse((r) => r.permits.deploy(ctx)) &&
        this.related.environment.traverse((e) => e.permits.deploy(ctx))) ||
      // always allow global admins
      this.related.environment.traverse((e) =>
        e.related.parent.traverse((g) => g.permits.admin(ctx))
      ),

    // read if you can read the release or environment (or explicit readers)
    read: (ctx: Context): boolean =>
      this.related.readers.includes(ctx.subject) ||
      this.related.release.traverse((r) => r.permits.read(ctx)) ||
      this.related.environment.traverse((e) => e.permits.read(ctx)),
  }
}
