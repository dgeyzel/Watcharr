/**
 * Routes only the signed in admin can use. Visitors are sent to the home page
 * from these, and failed auth on them goes to the admin login.
 * Every other page is public (visitors browse the owner's list read only).
 */
const ADMIN_ONLY_ROUTES = [
	"/discover",
	"/person",
	"/import",
	"/server",
	"/profile",
	"/manage_users",
];

export function isAdminOnlyRoute(pathname: string): boolean {
	return ADMIN_ONLY_ROUTES.some(
		(r) => pathname === r || pathname.startsWith(`${r}/`),
	);
}
