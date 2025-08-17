import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { ChevronsLeft, ChevronsRight } from "lucide-react";

type Props = {
  loading: boolean;
  total: number;
  page: number;
  pageCount: number;
  pageSize: number;
  setPage: (p: number) => void;
  setPageSize: (s: number) => void;
  startIdx: number;
  endIdx: number;
};

export default function PaginationControls({ loading, total, page, pageCount, pageSize, setPage, setPageSize, startIdx, endIdx }: Props) {
  if (loading || total <= 0) return null;

  const isFirstPage = page <= 1;
  const isLastPage = page >= pageCount || pageCount <= 1;
  const singlePage = pageCount <= 1;

  const leftLabel = singlePage
    ? `${total.toLocaleString()} results`
    : `${startIdx + 1}–${endIdx} of ${total.toLocaleString()}`;

  return (
    <div className="mt-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <div className="text-sm text-muted-foreground">{leftLabel}</div>
      <div className="flex items-center gap-3">
        <Select
          value={String(pageSize)}
          onValueChange={(v) => {
            setPageSize(parseInt(v));
            setPage(1);
          }}
        >
          <SelectTrigger className="w-[120px] shrink-0 whitespace-nowrap">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="25">25 / page</SelectItem>
            <SelectItem value="50">50 / page</SelectItem>
            <SelectItem value="100">100 / page</SelectItem>
          </SelectContent>
        </Select>
        {!singlePage && (
          <Pagination>
            <PaginationContent>
              <PaginationItem>
                <PaginationLink
                  href="#"
                  className={`gap-1 pl-2.5 ${isFirstPage ? "pointer-events-none opacity-50" : ""}`}
                  onClick={(e) => {
                    e.preventDefault();
                    if (isFirstPage) return;
                    setPage(1);
                  }}
                  aria-label="Go to first page"
                  aria-disabled={isFirstPage}
                  size="default"
                >
                  <ChevronsLeft className="h-4 w-4" />
                  <span>First</span>
                </PaginationLink>
              </PaginationItem>
              <PaginationItem>
                <PaginationPrevious
                  href="#"
                  className={isFirstPage ? "pointer-events-none opacity-50" : undefined}
                  onClick={(e) => {
                    e.preventDefault();
                    if (isFirstPage) return;
                    const prev = page - 1;
                    setPage(prev < 1 ? 1 : prev);
                  }}
                  aria-disabled={isFirstPage}
                />
              </PaginationItem>
              <PaginationItem>
                <PaginationLink href="#" isActive>
                  {page}
                </PaginationLink>
              </PaginationItem>
              <PaginationItem>
                <PaginationNext
                  href="#"
                  className={isLastPage ? "pointer-events-none opacity-50" : undefined}
                  onClick={(e) => {
                    e.preventDefault();
                    if (isLastPage) return;
                    const next = page + 1;
                    setPage(next > pageCount ? pageCount : next);
                  }}
                  aria-disabled={isLastPage}
                />
              </PaginationItem>
              <PaginationItem>
                <PaginationLink
                  href="#"
                  className={`gap-1 pr-2.5 ${isLastPage ? "pointer-events-none opacity-50" : ""}`}
                  onClick={(e) => {
                    e.preventDefault();
                    if (isLastPage) return;
                    setPage(pageCount);
                  }}
                  aria-label="Go to last page"
                  aria-disabled={isLastPage}
                  size="default"
                >
                  <span>Last</span>
                  <ChevronsRight className="h-4 w-4" />
                </PaginationLink>
              </PaginationItem>
            </PaginationContent>
          </Pagination>
        )}
      </div>
    </div>
  );
}
