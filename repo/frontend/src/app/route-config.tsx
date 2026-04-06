import type { ComponentType } from "react";

import type { Permission } from "../auth/policy";
import { AdminPage } from "../features/admin/AdminPage";
import { BookingFlowPage } from "../features/booking/BookingFlowPage";
import { CalendarPage } from "../features/calendar/CalendarPage";
import { ContentLibraryPage } from "../features/content/ContentLibraryPage";
import { DashboardPage } from "../features/dashboard/DashboardPage";
import { PlanningPage } from "../features/planning/PlanningPage";

export type AppRoute = {
  path: string;
  key: string;
  title: string;
  component: ComponentType;
  permission: Permission;
};

export const appRoutes: AppRoute[] = [
  {
    key: "admin",
    title: "Administrator",
    path: "/admin",
    component: AdminPage,
    permission: "admin.view",
  },
  {
    key: "dashboard",
    title: "Dashboard",
    path: "/dashboard",
    component: DashboardPage,
    permission: "dashboard.view",
  },
  {
    key: "calendar",
    title: "Calendar",
    path: "/calendar",
    component: CalendarPage,
    permission: "calendar.view",
  },
  {
    key: "booking",
    title: "Booking",
    path: "/booking",
    component: BookingFlowPage,
    permission: "booking.view",
  },
  {
    key: "content",
    title: "Content Library",
    path: "/content",
    component: ContentLibraryPage,
    permission: "content.view",
  },
  {
    key: "planning",
    title: "Tasks & Planning",
    path: "/planning",
    component: PlanningPage,
    permission: "planning.view",
  },
];
