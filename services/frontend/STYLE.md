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

### 6. Comments and Documentation
- Code should be self-documenting through clear naming
- Comments explain "why", not "what"
- Complex business logic deserves a comment explaining the intent
- Remove commented-out code - version control preserves history

### 7. Consistent Formatting
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