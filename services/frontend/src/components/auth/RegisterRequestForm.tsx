import { useEffect, useRef } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { withLatency } from "@/mocks/latency";
import { useAppStore } from "@/store/app-store";

const FREE_EMAIL_DOMAINS = new Set([
  "gmail.com",
  "yahoo.com",
  "outlook.com",
  "hotmail.com",
  "icloud.com",
  "aol.com",
  "proton.me",
  "protonmail.com",
  "mail.com",
  "yandex.com",
  "pm.me",
]);

const schema = z.object({
  email: z
    .string()
    .min(1, "Email is required")
    .email("Enter a valid email address")
    .refine((val) => {
      const domain = val.split("@")[1]?.toLowerCase();
      return Boolean(domain) && !FREE_EMAIL_DOMAINS.has(domain!);
    }, "Please use your work email (no free domains)")
});

export default function RegisterRequestForm({ onDone, defaultEmail, autoSubmit }: { onDone: (email: string) => void; defaultEmail?: string; autoSubmit?: boolean }) {
  const { actions } = useAppStore();
  const form = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { email: defaultEmail ?? "" },
  });

  const onSubmit = async (values: z.infer<typeof schema>) => {
    await withLatency();
    actions.addAudit({
      actor: values.email,
      action: "access.request",
      resource: "registration",
      meta: { email: values.email },
    });
    onDone(values.email);
  };

  const autoSubmittedRef = useRef(false);

  useEffect(() => {
    if (defaultEmail) {
      form.setValue("email", defaultEmail, { shouldValidate: false, shouldDirty: false });
    }
  }, [defaultEmail, form]);

  useEffect(() => {
    if (autoSubmit && defaultEmail && !autoSubmittedRef.current) {
      autoSubmittedRef.current = true;
      form.setValue("email", defaultEmail, { shouldValidate: false, shouldDirty: false });
      // Defer to ensure any dialog open animation completes
      Promise.resolve().then(() => form.handleSubmit(onSubmit)());
    }
  }, [autoSubmit, defaultEmail, form, onSubmit]);

  const submitting = form.formState.isSubmitting;

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4" noValidate>
        <FormField
          control={form.control}
          name="email"
          render={({ field }) => (
            <FormItem className="group">
              <FormLabel>Work email</FormLabel>
              <FormControl>
                <Input type="email" placeholder="you@company.com" autoFocus inputMode="email" {...field} />
              </FormControl>
              <FormDescription className="mt-3 text-[13px] leading-tight text-muted-foreground/90 opacity-90 transition-opacity duration-200 group-focus-within:opacity-100">
                We’ll email you a login link after approval.
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button type="submit" className="w-full hover-scale" disabled={submitting} aria-disabled={submitting}>
          {submitting ? "Submitting..." : "Request access"}
        </Button>
      </form>
    </Form>
  );
}
