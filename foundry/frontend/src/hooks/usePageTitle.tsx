import { Helmet } from "react-helmet-async";

export function usePageTitle(title: string, description?: string, path: string = "/") {
  return (
    <Helmet>
      <title>{title}</title>
      {description && <meta name="description" content={description} />}
      <link rel="canonical" href={path} />
    </Helmet>
  );
}
