import { createContext } from "react";
import type { SiteInfo } from "../types/api";

export const SiteInfoContext = createContext<SiteInfo | null>(null);
