/**
 * Spark Job Service
 * Handles batch Spark job CRUD and logs via Backend API
 */
import api from '../config/axios';

// ============ Types ============

export type SparkJobType = 'scala' | 'python';
export type SparkJobStatus =
  | 'queued'
  | 'submitted'
  | 'running'
  | 'success'
  | 'failed'
  | 'cancelled'
  | 'unknown';

export interface SparkJobResource {
  cpu: string;
  memory: string;
  executors: number;
}

export interface SparkJobDTO {
  id: string;
  name: string;
  type: SparkJobType;
  status: SparkJobStatus;
  main_class?: string;
  main_app_file?: string;
  arguments?: string;
  resources: SparkJobResource;
  user_id: number;
  created_at: string;
  updated_at: string;
  started_at?: string;
  finished_at?: string;
}

export interface SparkJobListResponse {
  items: SparkJobDTO[];
  total: number;
  page: number;
  page_size: number;
}

export interface SparkJobLogsResponse {
  logs: string;
  job_id: string;
}

export interface CreateSparkJobPayload {
  name: string;
  type: SparkJobType;
  main_class?: string;
  main_app_file?: string;
  arguments?: string;
  resources: SparkJobResource;
}

// ============ API Functions ============

const BASE_URL = '/api/v1/spark/jobs';

/**
 * List Spark batch jobs
 */
export async function listSparkJobs(
  page = 1,
  pageSize = 20,
): Promise<SparkJobListResponse> {
  try {
    const response = await api.get(BASE_URL, {
      params: { page, page_size: pageSize },
    });
    return response.data;
  } catch (error) {
    console.error('Failed to list spark jobs:', error);
    return { items: [], total: 0, page, page_size: pageSize };
  }
}

/**
 * Get details of a single Spark job
 */
export async function getSparkJob(jobId: string): Promise<SparkJobDTO | null> {
  try {
    const response = await api.get(`${BASE_URL}/${jobId}`, {
      params: { _t: Date.now() },
    });
    return response.data;
  } catch (error) {
    console.error('Failed to get spark job:', error);
    return null;
  }
}

/**
 * Create a new Spark batch job
 */
export async function createSparkJob(
  data: CreateSparkJobPayload,
): Promise<SparkJobDTO | null> {
  try {
    const response = await api.post(BASE_URL, data);
    return response.data;
  } catch (error) {
    console.error('Failed to create spark job:', error);
    return null;
  }
}

/**
 * Delete a Spark job
 */
export async function deleteSparkJob(jobId: string): Promise<boolean> {
  try {
    await api.delete(`${BASE_URL}/${jobId}`);
    return true;
  } catch (error) {
    console.error('Failed to delete spark job:', error);
    return false;
  }
}

/**
 * Stop (cancel) a running Spark job
 */
export async function stopSparkJob(jobId: string): Promise<boolean> {
  try {
    await api.delete(`${BASE_URL}/${jobId}`);
    return true;
  } catch (error) {
    console.error('Failed to stop spark job:', error);
    return false;
  }
}

/**
 * Get logs for a Spark job
 */
export async function getSparkJobLogs(
  jobId: string,
): Promise<SparkJobLogsResponse | null> {
  try {
    const response = await api.get(`${BASE_URL}/${jobId}/logs`, {
      params: { _t: Date.now() },
    });
    return response.data;
  } catch (error) {
    console.error('Failed to get spark job logs:', error);
    return null;
  }
}

// ============ Export ============

export const sparkJobService = {
  listSparkJobs,
  getSparkJob,
  createSparkJob,
  deleteSparkJob,
  stopSparkJob,
  getSparkJobLogs,
};

export default sparkJobService;
