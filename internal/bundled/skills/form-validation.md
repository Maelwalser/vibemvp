# Form Validation Skill Guide

## React Hook Form

### Basic Usage

```tsx
import { useForm, SubmitHandler } from 'react-hook-form'

interface LoginForm {
  email: string
  password: string
  rememberMe: boolean
}

function LoginForm() {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginForm>({ defaultValues: { rememberMe: false } })

  const onSubmit: SubmitHandler<LoginForm> = async (data) => {
    await fetch('/api/login', { method: 'POST', body: JSON.stringify(data) })
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <input
        {...register('email', {
          required: 'Email is required',
          pattern: { value: /\S+@\S+\.\S+/, message: 'Invalid email' },
        })}
      />
      {errors.email && <span>{errors.email.message}</span>}

      <input
        type="password"
        {...register('password', {
          required: 'Password is required',
          minLength: { value: 8, message: 'Minimum 8 characters' },
        })}
      />
      {errors.password && <span>{errors.password.message}</span>}

      <button type="submit" disabled={isSubmitting}>
        {isSubmitting ? 'Logging in...' : 'Login'}
      </button>
    </form>
  )
}
```

### Controller for Controlled Inputs

```tsx
import { useForm, Controller } from 'react-hook-form'
import Select from 'react-select'
import DatePicker from 'react-datepicker'

function BookingForm() {
  const { control, handleSubmit } = useForm<BookingForm>()

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <Controller
        name="category"
        control={control}
        rules={{ required: 'Please select a category' }}
        render={({ field, fieldState }) => (
          <>
            <Select
              {...field}
              options={categoryOptions}
              onChange={(opt) => field.onChange(opt?.value)}
            />
            {fieldState.error && <span>{fieldState.error.message}</span>}
          </>
        )}
      />

      <Controller
        name="date"
        control={control}
        render={({ field }) => (
          <DatePicker selected={field.value} onChange={field.onChange} />
        )}
      />
    </form>
  )
}
```

### watch for Dependent Fields

```tsx
function ShippingForm() {
  const { register, watch } = useForm<ShippingForm>()
  const country = watch('country')

  return (
    <form>
      <select {...register('country')}>
        <option value="US">United States</option>
        <option value="CA">Canada</option>
      </select>

      {country === 'US' && (
        <input {...register('zipCode', { required: true, pattern: /^\d{5}$/ })} placeholder="ZIP" />
      )}
      {country === 'CA' && (
        <input {...register('postalCode', { required: true })} placeholder="Postal Code" />
      )}
    </form>
  )
}
```

### useFieldArray for Dynamic Fields

```tsx
import { useForm, useFieldArray } from 'react-hook-form'

interface InvoiceForm {
  lineItems: { description: string; amount: number }[]
}

function InvoiceForm() {
  const { register, control, handleSubmit } = useForm<InvoiceForm>({
    defaultValues: { lineItems: [{ description: '', amount: 0 }] },
  })
  const { fields, append, remove } = useFieldArray({ control, name: 'lineItems' })

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      {fields.map((field, index) => (
        <div key={field.id}>
          <input {...register(`lineItems.${index}.description`, { required: true })} />
          <input
            type="number"
            {...register(`lineItems.${index}.amount`, { valueAsNumber: true })}
          />
          <button type="button" onClick={() => remove(index)}>Remove</button>
        </div>
      ))}
      <button type="button" onClick={() => append({ description: '', amount: 0 })}>
        Add Line Item
      </button>
    </form>
  )
}
```

### With Zod Resolver

```tsx
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'

const schema = z.object({
  email: z.string().email('Invalid email'),
  age: z.number().min(18, 'Must be 18+').max(120),
  role: z.enum(['admin', 'user', 'viewer']),
})

type FormData = z.infer<typeof schema>

function MyForm() {
  const { register, handleSubmit, formState: { errors } } = useForm<FormData>({
    resolver: zodResolver(schema),
  })
  // ...
}
```

---

## Formik

```tsx
import { Formik, Form, Field, ErrorMessage, FieldArray } from 'formik'
import * as Yup from 'yup'

const validationSchema = Yup.object({
  name: Yup.string().min(2, 'Too short').required('Required'),
  email: Yup.string().email('Invalid email').required('Required'),
  friends: Yup.array().of(Yup.string().required('Friend name required')),
})

function ProfileForm() {
  return (
    <Formik
      initialValues={{ name: '', email: '', friends: [''] }}
      validationSchema={validationSchema}
      onSubmit={async (values, { setSubmitting }) => {
        await saveProfile(values)
        setSubmitting(false)
      }}
    >
      {({ values, isSubmitting }) => (
        <Form>
          <Field name="name" placeholder="Name" />
          <ErrorMessage name="name" component="span" />

          <Field name="email" type="email" placeholder="Email" />
          <ErrorMessage name="email" component="span" />

          <FieldArray name="friends">
            {({ push, remove }) => (
              <div>
                {values.friends.map((_, i) => (
                  <div key={i}>
                    <Field name={`friends.${i}`} />
                    <button type="button" onClick={() => remove(i)}>Remove</button>
                  </div>
                ))}
                <button type="button" onClick={() => push('')}>Add Friend</button>
              </div>
            )}
          </FieldArray>

          <button type="submit" disabled={isSubmitting}>Save</button>
        </Form>
      )}
    </Formik>
  )
}
```

---

## Zod Schema Validation

```ts
import { z } from 'zod'

// Object schema
const UserSchema = z.object({
  id: z.string().uuid(),
  name: z.string().min(1).max(100),
  email: z.string().email(),
  age: z.number().int().min(0).max(150).optional(),
  role: z.enum(['admin', 'user', 'viewer']).default('user'),
  tags: z.array(z.string()).max(10),
  address: z.object({
    street: z.string(),
    city: z.string(),
    country: z.string().length(2),
  }).optional(),
  createdAt: z.coerce.date(),
})

type User = z.infer<typeof UserSchema>

// parse (throws ZodError) vs safeParse (returns result)
const result = UserSchema.safeParse(untrustedInput)
if (!result.success) {
  const errors = result.error.flatten().fieldErrors
  // { email: ['Invalid email'], name: ['String must contain at least 1 character'] }
  return { error: errors }
}
const user = result.data

// Transformations
const TrimmedString = z.string().trim().min(1)

// Refinements
const PasswordSchema = z.string()
  .min(8)
  .refine(s => /[A-Z]/.test(s), 'Needs uppercase')
  .refine(s => /[0-9]/.test(s), 'Needs number')

// Superrefine for cross-field validation
const SignUpSchema = z.object({
  password: z.string().min(8),
  confirmPassword: z.string(),
}).superRefine(({ password, confirmPassword }, ctx) => {
  if (password !== confirmPassword) {
    ctx.addIssue({ code: 'custom', message: 'Passwords do not match', path: ['confirmPassword'] })
  }
})
```

---

## Valibot (Lightweight Alternative to Zod)

```ts
import * as v from 'valibot'

const UserSchema = v.object({
  name: v.pipe(v.string(), v.minLength(1), v.maxLength(100)),
  email: v.pipe(v.string(), v.email()),
  age: v.optional(v.pipe(v.number(), v.integer(), v.minValue(0))),
})

type User = v.InferOutput<typeof UserSchema>

const result = v.safeParse(UserSchema, untrustedInput)
if (!result.success) {
  console.error(result.issues)
}
```

---

## class-validator (Node.js / NestJS)

```ts
import { IsString, IsEmail, IsInt, Min, Max, ValidateNested, IsArray } from 'class-validator'
import { Type } from 'class-transformer'
import { validate } from 'class-validator'

class AddressDto {
  @IsString()
  street: string

  @IsString()
  city: string
}

class CreateUserDto {
  @IsString()
  name: string

  @IsEmail()
  email: string

  @IsInt()
  @Min(18)
  @Max(120)
  age: number

  @ValidateNested()
  @Type(() => AddressDto)
  address: AddressDto
}

// In a service or middleware
async function validateInput(input: unknown) {
  const dto = Object.assign(new CreateUserDto(), input)
  const errors = await validate(dto)
  if (errors.length > 0) {
    throw new Error(errors.map(e => Object.values(e.constraints ?? {})).flat().join(', '))
  }
  return dto
}
```

---

## Vee-Validate (Vue)

```vue
<script setup lang="ts">
import { useForm, defineField } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'

const schema = toTypedSchema(z.object({
  email: z.string().email(),
  password: z.string().min(8),
}))

const { handleSubmit, errors } = useForm({ validationSchema: schema })

const [email, emailAttrs] = defineField('email')
const [password, passwordAttrs] = defineField('password')

const onSubmit = handleSubmit(async (values) => {
  await login(values)
})
</script>

<template>
  <form @submit="onSubmit">
    <input v-model="email" v-bind="emailAttrs" type="email" />
    <span>{{ errors.email }}</span>

    <input v-model="password" v-bind="passwordAttrs" type="password" />
    <span>{{ errors.password }}</span>

    <button type="submit">Login</button>
  </form>
</template>
```

---

## Key Rules

- React Hook Form: use `zodResolver` to co-locate schema and types — avoid duplicate validation logic.
- Always use `handleSubmit` wrapper to prevent native form submission and ensure validation runs.
- `useFieldArray` keys must use `field.id` (not array index) to avoid remount bugs.
- Zod: use `safeParse` in API handlers to get structured errors without try/catch.
- class-validator: pair with `class-transformer` (`@Type`) for nested object validation.
- Never trust client-side validation alone — always re-validate on the server.
