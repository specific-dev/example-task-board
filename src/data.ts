export type TaskStatus = "todo" | "in-progress" | "done"

export interface Tag {
  label: string
  color: string
}

export interface Task {
  id: string
  title: string
  description: string
  status: TaskStatus
  tags: Tag[]
}

export const columns: { id: TaskStatus; title: string }[] = [
  { id: "todo", title: "To Do" },
  { id: "in-progress", title: "In Progress" },
  { id: "done", title: "Done" },
]

export const tasks: Task[] = [
  {
    id: "1",
    title: "Design system tokens",
    description: "Define color palette, typography scale, and spacing tokens for the component library.",
    status: "todo",
    tags: [
      { label: "Design", color: "#7c3aed" },
      { label: "Foundation", color: "#0891b2" },
    ],
  },
  {
    id: "2",
    title: "Set up CI pipeline",
    description: "Configure GitHub Actions for linting, type checking, and running tests on every PR.",
    status: "todo",
    tags: [
      { label: "DevOps", color: "#ea580c" },
    ],
  },
  {
    id: "3",
    title: "User authentication flow",
    description: "Implement sign-up, login, and password reset screens with form validation.",
    status: "todo",
    tags: [
      { label: "Feature", color: "#2563eb" },
      { label: "Auth", color: "#dc2626" },
    ],
  },
  {
    id: "4",
    title: "API rate limiting",
    description: "Add middleware to enforce per-user rate limits on all public API endpoints.",
    status: "in-progress",
    tags: [
      { label: "Backend", color: "#16a34a" },
      { label: "Security", color: "#dc2626" },
    ],
  },
  {
    id: "5",
    title: "Dashboard layout",
    description: "Build the responsive grid layout for the main dashboard with sidebar navigation.",
    status: "in-progress",
    tags: [
      { label: "Feature", color: "#2563eb" },
      { label: "UI", color: "#7c3aed" },
    ],
  },
  {
    id: "6",
    title: "Database indexing",
    description: "Optimize slow queries by adding composite indexes on frequently filtered columns.",
    status: "done",
    tags: [
      { label: "Backend", color: "#16a34a" },
      { label: "Performance", color: "#ca8a04" },
    ],
  },
  {
    id: "7",
    title: "Onboarding tooltip tour",
    description: "Create a guided tooltip walkthrough for first-time users after sign-up.",
    status: "done",
    tags: [
      { label: "Feature", color: "#2563eb" },
      { label: "UX", color: "#7c3aed" },
    ],
  },
]
