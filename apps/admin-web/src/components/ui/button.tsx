import { cva, type VariantProps } from "class-variance-authority";
import type { ButtonHTMLAttributes } from "react";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex h-9 items-center justify-center gap-2 rounded-md px-3 text-sm font-medium transition disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        primary: "bg-[#2563eb] text-white hover:bg-[#1d4ed8]",
        outline: "border border-[#d8dde8] bg-white text-[#1f2430] hover:bg-[#f6f8fb]",
        ghost: "text-[#4b5565] hover:bg-[#eef2f7]",
      },
      size: {
        sm: "h-8 px-2.5 text-xs",
        md: "h-9 px-3",
      },
    },
    defaultVariants: {
      variant: "primary",
      size: "md",
    },
  },
);

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> &
  VariantProps<typeof buttonVariants>;

export function Button({ className, variant, size, ...props }: ButtonProps) {
  return <button className={cn(buttonVariants({ variant, size }), className)} {...props} />;
}
