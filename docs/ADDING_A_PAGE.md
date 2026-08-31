# Adding a New Page

When creating a new page or section, update **all** of the following:

1. **OG tags** - `internal/og/og.go`: add path matching in `metaForPath()` and a meta method for detail pages. Canonical URL, og:title, og:description, og:image, and twitter:* tags are auto-injected from the returned `Meta`.
2. **Admin Content Rules** - `frontend/src/pages/admin/AdminContentRules.tsx`: add to the `pages` array with a `rules_<page_name>` key, and register the matching `SettingRules...` in `internal/config/config.go`. Declaring the `SiteSettingDef` is only half of it: it must also go into the `AllSiteSettings` slice, or the key is never persisted or served.
3. **Sidebar** - `frontend/src/components/layout/Sidebar/Sidebar.tsx`: add `<NavLink>` in the appropriate section.
4. **Profile settings default page** - `frontend/src/pages/profile/SettingsPage.tsx`: add `<option>` to the Home Page dropdown.
5. **Lazy page export** - `frontend/src/pages/lazyPages.ts`: export the page through the `named()` adapter. `App.tsx` imports every routed page from here, never from the page file directly, so a page missing from this module cannot be routed.
6. **Home page routes** - `frontend/src/App.tsx`: add to the `homePageRoutes` object and add a `<Route>` element.
7. **Backend routes** - `internal/controllers/`: routes are never registered inline. Add a `getAllXRoutes() []FSetupRoute` method returning one `setupX(r fiber.Router)` method per endpoint, then append that list to `GetAPIRoutes()` (mounted under `/api/v1`) or `GetPageRoutes()` (mounted at the root) in `internal/controllers/service.go`. `internal/routes/public_routes.go` walks both lists and needs no change.
8. **Sitemap** - `internal/sitemap/service.go`: add the URL to the `staticPaths` slice. For a collection, add a `Service` method that builds its `Entry` list, add the sub-sitemap suffix to `IndexEntries()`, and wire a handler plus route in `internal/controllers/sitemap_controller.go`.
9. **Content filter rules** - `internal/contentfilter`: if the new page accepts user text, make sure its service runs input through the content filter pipeline.
10. **Search** - if the new page introduces a searchable entity, add a `SearchSource` to `searchSources` in `internal/repository/search.go` and a matching URL builder to `urlBuilders` in `internal/search/urls.go`. An `init()` in `urls.go` panics at startup if you register one without the other.
