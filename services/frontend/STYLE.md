# Code Style Guide

## Core Principle
**Code is written for humans to read, not just for computers to execute.**

Every line of code should be immediately understandable to a developer reading it for the first time. Clarity and readability always take precedence over cleverness or brevity.

## Readability Standards

### 1. One Concept Per Line
- Each line should express a single, clear idea
- Complex operations must be broken down into named steps
- Avoid chaining multiple operations on a single line

**Bad:**
```typescript
const result = (data as unknown as { nested?: { value?: string } }).nested?.value || defaultValue;
```

**Good:**
```typescript
const typedData = data as unknown as NestedData;
const nestedValue = typedData.nested?.value;
const result = nestedValue || defaultValue;
```

### 2. Meaningful Variable Names
- Use descriptive names that explain the purpose
- Avoid single-letter variables except in simple loops
- Name intermediate results to document the transformation flow

**Bad:**
```typescript
const res = await api.call();
const d = res.data as SomeType;
```

**Good:**
```typescript
const apiResponse = await api.call();
const userData = apiResponse.data as UserData;
```

### 3. Explicit Type Handling
- Define clear type interfaces instead of inline type assertions
- Separate type casting from business logic
- Document why type assertions are necessary when unavoidable

**Bad:**
```typescript
const value = ((response as any).body as { data: { items: Item[] } }).data.items[0];
```

**Good:**
```typescript
interface ApiResponse {
  body: {
    data: {
      items: Item[];
    };
  };
}

const typedResponse = response as ApiResponse;
const items = typedResponse.body.data.items;
const firstItem = items[0];
```

### 4. Clear Control Flow
- Each step in a process should be visually distinct
- Error handling should be separate from happy path logic
- Use early returns to reduce nesting
- Add whitespace between logical sections

**Bad:**
```typescript
try {
  const res = await fetch(url); if (!res.ok) throw new Error('Failed'); return res.json();
} catch (e) { console.error(e); return null; }
```

**Good:**
```typescript
try {
  const response = await fetch(url);
  
  if (!response.ok) {
    throw new Error('Failed to fetch data');
  }
  
  return response.json();
} catch (error) {
  console.error('Error fetching data:', error);
  return null;
}
```

### 5. Function and Method Decomposition
- Functions should do one thing well
- Extract complex conditions into named boolean functions
- Break long functions into smaller, named helper functions

**Bad:**
```typescript
if (user.role === 'admin' || (user.role === 'moderator' && user.permissions.includes('edit')) || user.id === resource.ownerId) {
  // allow action
}
```

**Good:**
```typescript
function canEditResource(user: User, resource: Resource): boolean {
  const isAdmin = user.role === 'admin';
  const isModeratorWithEditPermission = user.role === 'moderator' && user.permissions.includes('edit');
  const isOwner = user.id === resource.ownerId;
  
  return isAdmin || isModeratorWithEditPermission || isOwner;
}

if (canEditResource(user, resource)) {
  // allow action
}
```

### 6. Code Organization and Reuse
- **Always check `src/lib` BEFORE writing new code** to avoid duplication
- Move reusable helper functions to appropriate modules in `src/lib`
- Never duplicate utility functions across components or pages
- Organize shared code by domain (e.g., `lib/auth/`, `lib/api/`, `lib/utils/`)

**Bad:**
```typescript
// In Profile.tsx
function formatDate(date: Date): string {
  return new Intl.DateTimeFormat('en-US').format(date);
}

// In Dashboard.tsx (duplicated)
function formatDate(date: Date): string {
  return new Intl.DateTimeFormat('en-US').format(date);
}
```

**Good:**
```typescript
// In lib/utils/date.ts
export function formatDate(date: Date): string {
  return new Intl.DateTimeFormat('en-US').format(date);
}

// In Profile.tsx
import { formatDate } from '@/lib/utils/date';

// In Dashboard.tsx
import { formatDate } from '@/lib/utils/date';
```

**Code Organization Checklist:**
1. Before implementing any utility function, search `src/lib` for existing implementations
2. If a function is used in more than one file, it belongs in `src/lib`
3. Group related utilities together in domain-specific modules
4. Document exported functions with JSDoc comments
5. Keep page/component files focused on UI logic, not utility functions

### 7. Comments and Documentation
- Code should be self-documenting through clear naming
- Comments explain "why", not "what"
- Complex business logic deserves a comment explaining the intent
- Remove commented-out code - version control preserves history

### 8. Consistent Formatting
- Use consistent indentation (2 spaces for TypeScript/JavaScript)
- Add blank lines between logical sections
- Group related declarations together
- Align similar operations for visual scanning

## React/TypeScript Specific Guidelines

### Component Structure
- Props interfaces should be clearly defined and exported
- Destructure props at the component level for clarity
- Separate business logic from render logic
- Extract complex JSX into named sub-components or functions

### Hooks and State
- Group related state declarations together
- Custom hooks should have clear, descriptive names
- Effects should have clear dependencies and cleanup

### Type Safety
- Prefer explicit types over 'any'
- Use proper generics instead of type assertions where possible
- Define domain types in dedicated type files

### OpenAPI Generated Types (CRITICAL)
**ALWAYS use the types generated from the OpenAPI specification. NEVER bypass or cast around them.**

#### Mandatory Rules for Generated Types
1. **Use generated types from `forge-client` for ALL API interactions**
2. **NEVER cast or bypass generated types** - they are the source of truth
3. **If generated types are wrong, FIX THE API** - update Swagger definitions in the backend
4. **NO WORKAROUNDS** - incorrect types indicate an API contract violation

#### Examples

**ABSOLUTELY FORBIDDEN:**
```typescript
// NEVER DO THIS - bypassing generated types
const response = await forge.GET('/api/users');
const users = response.data as any; // ❌ FORBIDDEN

// NEVER DO THIS - casting around incorrect types
const data = response.data as unknown as MyCustomType; // ❌ FORBIDDEN

// NEVER DO THIS - creating duplicate type definitions
interface User { // ❌ FORBIDDEN if this exists in generated types
  id: string;
  name: string;
}
```

**CORRECT APPROACH:**
```typescript
import type { components } from 'forge-client';

// Use the generated types directly
type User = components['schemas']['User'];
type GetUsersResponse = components['schemas']['GetUsersResponse'];

const response = await forge.GET('/api/users');
if (response.data) {
  // response.data is already correctly typed from OpenAPI
  const users = response.data; // ✅ CORRECT
}
```

#### When Types Don't Match Reality

If the generated types don't match the actual API response:

1. **STOP** - Do not proceed with workarounds
2. **FIX THE API** - Update the Swagger/OpenAPI definitions in the backend
3. **REGENERATE** - Run the type generation to get updated types
4. **VERIFY** - Ensure the types now match the actual API behavior

**The Process:**
```bash
# 1. Fix the API swagger definitions (in ../api/*)
# 2. Regenerate the client types
npm run generate:client  # or appropriate command
# 3. Use the updated types in your code
```

#### Why This Matters
- **Type safety** - The OpenAPI spec IS the contract
- **Consistency** - One source of truth for API types
- **Maintainability** - Changes to API are automatically reflected
- **Documentation** - Generated types serve as living documentation
- **Debugging** - Type mismatches catch API breaking changes immediately

**Remember:** If you're tempted to cast around generated types, you're identifying a bug in the API specification that MUST be fixed at the source.

## Code Quality Enforcement

### All Code Changes MUST Pass Quality Checks
**Code changes are NOT complete until all formatting and linting checks pass.**

Before considering any code change complete, you MUST run and pass:
```bash
npm run check-all
```

This ensures:
- **TypeScript compilation** succeeds (`npm run typecheck`)
- **ESLint** passes with no errors (`npm run lint`)
- **Prettier formatting** is applied (`npm run format:check`)

#### Quick Fix Commands
```bash
# Fix all issues automatically
npm run fix-all

# Individual fix commands
npm run format      # Fix formatting issues
npm run lint:fix    # Fix auto-fixable lint issues
```

#### Pre-Commit Checklist
1. ✅ Run `npm run check-all` - must pass with no errors
2. ✅ Fix any ESLint errors (warnings are acceptable but should be minimized)
3. ✅ Apply Prettier formatting to all changed files
4. ✅ Ensure TypeScript compilation succeeds

**Important:** Code with formatting or linting errors is considered broken code. Always run checks before completing any task.

## The Refactoring Test
When reviewing or refactoring code, ask yourself:
1. Can a new developer understand this code in under 30 seconds?
2. Is the intent of each line immediately clear?
3. Are the steps in the process easy to follow?
4. Could I explain this code to someone over the phone?

If the answer to any of these is "no", the code needs to be simplified.

## Remember
- **Readability > Cleverness**
- **Clarity > Brevity**
- **Explicit > Implicit**
- **Simple > Complex**

Good code reads like well-written prose - it tells a clear story of what it does and why.