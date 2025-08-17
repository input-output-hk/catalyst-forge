import type { User } from "@/features/users/types";

const randomFrom = <T>(arr: T[]) => arr[Math.floor(Math.random() * arr.length)];
const makeId = (prefix: string) => `${prefix}_${Math.random().toString(36).slice(2, 9)}`;

export function seedUsers(count = 96): User[] {
  const firstNames = [
    "Alice",
    "Bob",
    "Carol",
    "David",
    "Eve",
    "Frank",
    "Grace",
    "Heidi",
    "Ivan",
    "Judy",
    "Mallory",
    "Nia",
    "Olivia",
    "Peggy",
    "Rupert",
    "Sybil",
    "Trent",
    "Victor",
    "Walter",
    "Yara",
    "Zoe",
  ];
  const lastNames = [
    "Anderson",
    "Brown",
    "Clark",
    "Davis",
    "Evans",
    "Foster",
    "Garcia",
    "Harris",
    "Iverson",
    "Johnson",
    "Klein",
    "Lopez",
    "Miller",
    "Nguyen",
    "Olsen",
    "Patel",
    "Quinn",
    "Roberts",
    "Smith",
    "Turner",
    "Ulrich",
    "Vega",
    "Williams",
    "Xu",
    "Young",
    "Zimmerman",
  ];
  const roles: Array<User["role"]> = ["admin", "member"];
  const statuses: Array<User["status"]> = [
    "active",
    "disabled",
    "pending_invite",
    "pending_approval",
  ];
  const users: User[] = [];
  const now = Date.now();

  for (let i = 0; i < count; i++) {
    const fname = randomFrom(firstNames);
    const lname = randomFrom(lastNames);
    const name = `${fname} ${lname}`;
    const email = `${fname}.${lname}${i % 7 === 0 ? ".test" : ""}@example.com`.toLowerCase();
    const role = Math.random() < 0.18 ? "admin" : "member";
    const status = randomFrom(statuses);
    const createdAt = new Date(
      now - Math.floor(Math.random() * 1000 * 60 * 60 * 24 * 365)
    ).toISOString();
    const lastActivityAt =
      Math.random() < 0.12
        ? undefined
        : new Date(now - Math.floor(Math.random() * 1000 * 60 * 60 * 24 * 120)).toISOString();
    const sessions = status === "disabled" ? 0 : Math.floor(Math.random() * 4);

    users.push({
      id: makeId("usr"),
      name,
      email,
      role,
      status,
      createdAt,
      lastActivityAt,
      sessions,
      credentials: [],
      invites: null,
      requests: [],
    });
  }
  return users.sort((a, b) => a.name.localeCompare(b.name));
}
