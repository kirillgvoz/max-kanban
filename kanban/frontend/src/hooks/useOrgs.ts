import { useState, useEffect, useCallback } from "react";
import { api } from "../api/client";
import type { Organization, OrgDetail } from "../types";

export function useOrgs() {
  const [orgs, setOrgs] = useState<Organization[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchOrgs = useCallback(async () => {
    try {
      const data = await api.orgs.list();
      setOrgs(data);
    } catch (e) {
      console.error("Failed to fetch orgs:", e);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchOrgs();
  }, [fetchOrgs]);

  const createOrg = useCallback(async (name: string) => {
    const org = await api.orgs.create(name);
    setOrgs((prev) => [...prev, org]);
    return org;
  }, []);

  return { orgs, loading, createOrg, refetch: fetchOrgs };
}

export function useOrg(id: number | null) {
  const [org, setOrg] = useState<OrgDetail | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchOrg = useCallback(async () => {
    if (!id) return;
    try {
      const data = await api.orgs.get(id);
      setOrg(data);
    } catch (e) {
      console.error("Failed to fetch org:", e);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    fetchOrg();
  }, [fetchOrg]);

  return { org, loading, refetch: fetchOrg };
}
