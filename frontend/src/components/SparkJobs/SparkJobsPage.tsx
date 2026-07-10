/**
 * SparkJobsPage - Manage batch Spark jobs
 */
'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import {
  Play,
  Square,
  Eye,
  Loader2,
  Sparkles,
  Plus,
  FileCode,
  Clock,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  sparkJobService,
  SparkJobDTO,
  SparkJobType,
  CreateSparkJobPayload,
} from '@/services/sparkJobService';

// ─── Helpers ────────────────────────────────────────────────────────────────

const STATUS_VARIANTS: Record<string, string> = {
  queued: 'bg-yellow-500/15 text-yellow-600 border-yellow-300',
  submitted: 'bg-blue-500/15 text-blue-600 border-blue-300',
  running: 'bg-emerald-500/15 text-emerald-600 border-emerald-300',
  success: 'bg-green-500/15 text-green-600 border-green-300',
  failed: 'bg-red-500/15 text-red-600 border-red-300',
  cancelled: 'bg-slate-500/15 text-slate-600 border-slate-300',
  unknown: 'bg-gray-500/15 text-gray-600 border-gray-300',
};

function jobStatusBadge(status: string) {
  const variant = STATUS_VARIANTS[status] || STATUS_VARIANTS.unknown;
  return (
    <Badge variant="outline" className={cn('capitalize font-medium', variant)}>
      {status}
    </Badge>
  );
}

function formatDateTime(iso: string | undefined | null) {
  if (!iso) return '—';
  return new Date(iso).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function formatDuration(start?: string, end?: string) {
  if (!start) return '—';
  const s = new Date(start).getTime();
  const e = end ? new Date(end).getTime() : Date.now();
  const ms = e - s;
  if (ms < 1000) return '<1s';
  const sec = Math.floor(ms / 1000);
  if (sec < 60) return `${sec}s`;
  const min = Math.floor(sec / 60);
  const rem = sec % 60;
  return `${min}m ${rem}s`;
}

const STOPPABLE_STATUSES = new Set(['queued', 'submitted', 'running']);

// ─── Default resource values ────────────────────────────────────────────────

const DEFAULT_RESOURCES = { cpu: '1', memory: '1g', executors: 1 };

// ─── Component ──────────────────────────────────────────────────────────────

export default function SparkJobsPage() {
  // ── List state ──
  const [jobs, setJobs] = useState<SparkJobDTO[]>([]);
  const [loading, setLoading] = useState(true);

  const loadJobs = useCallback(async () => {
    setLoading(true);
    const res = await sparkJobService.listSparkJobs(1, 50);
    setJobs(res.items);
    setLoading(false);
  }, []);

  useEffect(() => {
    loadJobs();
  }, [loadJobs]);

  // Auto-refresh every 10 s if any job is still running/queued/submitted
  useEffect(() => {
    const hasActive = jobs.some((j) =>
      ['queued', 'submitted', 'running'].includes(j.status),
    );
    if (!hasActive) return;
    const interval = window.setInterval(loadJobs, 10_000);
    return () => window.clearInterval(interval);
  }, [jobs, loadJobs]);

  // ── Create dialog state ──
  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});

  const [form, setForm] = useState<{
    name: string;
    type: SparkJobType;
    mainClass: string;
    mainAppFile: string;
    arguments: string;
    cpu: string;
    memory: string;
    executors: number;
  }>({
    name: '',
    type: 'scala',
    mainClass: '',
    mainAppFile: '',
    arguments: '',
    cpu: DEFAULT_RESOURCES.cpu,
    memory: DEFAULT_RESOURCES.memory,
    executors: DEFAULT_RESOURCES.executors,
  });

  const resetForm = () => {
    setForm({
      name: '',
      type: 'scala',
      mainClass: '',
      mainAppFile: '',
      arguments: '',
      cpu: DEFAULT_RESOURCES.cpu,
      memory: DEFAULT_RESOURCES.memory,
      executors: DEFAULT_RESOURCES.executors,
    });
    setFormErrors({});
  };

  const openCreate = () => {
    resetForm();
    setCreateOpen(true);
  };

  const validateForm = (): boolean => {
    const errors: Record<string, string> = {};
    if (!form.name.trim()) errors.name = 'Name is required';
    if (form.type === 'scala' && !form.mainClass.trim())
      errors.mainClass = 'Main class is required for Scala jobs';
    if (!form.mainAppFile.trim())
      errors.mainAppFile = 'Main app file is required';
    if (!form.cpu.trim()) errors.cpu = 'CPU is required';
    if (!form.memory.trim()) errors.memory = 'Memory is required';
    if (form.executors < 1) errors.executors = 'At least 1 executor';
    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleCreate = async () => {
    if (!validateForm()) return;
    setCreating(true);
    const payload: CreateSparkJobPayload = {
      name: form.name.trim(),
      type: form.type,
      main_class: form.type === 'scala' ? form.mainClass.trim() : undefined,
      main_app_file: form.mainAppFile.trim(),
      arguments: form.arguments.trim() || undefined,
      resources: {
        cpu: form.cpu.trim(),
        memory: form.memory.trim(),
        executors: form.executors,
      },
    };
    const job = await sparkJobService.createSparkJob(payload);
    setCreating(false);
    if (job) {
      setCreateOpen(false);
      setJobs((prev) => [job, ...prev]);
    }
  };

  // ── Logs dialog ──
  const [logsJob, setLogsJob] = useState<SparkJobDTO | null>(null);
  const [logsContent, setLogsContent] = useState<string>('');
  const [logsLoading, setLogsLoading] = useState(false);

  const openLogs = async (job: SparkJobDTO) => {
    setLogsJob(job);
    setLogsLoading(true);
    setLogsContent('');
    const res = await sparkJobService.getSparkJobLogs(job.id);
    setLogsContent(res?.logs ?? 'No logs available.');
    setLogsLoading(false);
  };

  // ── Stop confirmation ──
  const [stopTarget, setStopTarget] = useState<SparkJobDTO | null>(null);
  const [stopping, setStopping] = useState(false);

  const confirmStop = async () => {
    if (!stopTarget) return;
    setStopping(true);
    const ok = await sparkJobService.stopSparkJob(stopTarget.id);
    setStopping(false);
    if (ok) {
      setJobs((prev) =>
        prev.map((j) =>
          j.id === stopTarget.id ? { ...j, status: 'cancelled' as const } : j,
        ),
      );
    }
    setStopTarget(null);
  };

  // ── Render ──
  return (
    <div className="space-y-6">
      {/* ── Header ── */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Sparkles className="h-6 w-6" /> Spark Jobs
          </h1>
          <p className="text-muted-foreground">
            Submit and monitor batch Spark jobs
          </p>
        </div>

        <Button onClick={openCreate}>
          <Plus className="h-4 w-4 mr-2" />
          New Job
        </Button>
      </div>

      <Separator />

      {/* ── Loading ── */}
      {loading && (
        <div className="flex items-center justify-center h-48">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      )}

      {/* ── Empty state ── */}
      {!loading && jobs.length === 0 && (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center h-64">
            <FileCode className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground mb-4">No Spark jobs yet</p>
            <Button onClick={openCreate}>
              <Plus className="h-4 w-4 mr-2" />
              Submit your first job
            </Button>
          </CardContent>
        </Card>
      )}

      {/* ── Job table ── */}
      {!loading && jobs.length > 0 && (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Duration</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {jobs.map((job) => (
                  <TableRow key={job.id}>
                    <TableCell className="font-medium max-w-[200px] truncate">
                      {job.name}
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant="outline"
                        className={cn(
                          'capitalize font-mono text-xs',
                          job.type === 'scala'
                            ? 'bg-purple-500/10 text-purple-600 border-purple-300'
                            : 'bg-blue-500/10 text-blue-600 border-blue-300',
                        )}
                      >
                        {job.type}
                      </Badge>
                    </TableCell>
                    <TableCell>{jobStatusBadge(job.status)}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      <span className="flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        {formatDuration(job.started_at, job.finished_at)}
                      </span>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {formatDateTime(job.created_at)}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          title="View logs"
                          onClick={() => openLogs(job)}
                        >
                          <Eye className="h-4 w-4" />
                        </Button>
                        {STOPPABLE_STATUSES.has(job.status) && (
                          <Button
                            variant="ghost"
                            size="icon"
                            className="text-destructive hover:text-destructive"
                            title="Stop job"
                            onClick={() => setStopTarget(job)}
                          >
                            <Square className="h-4 w-4" />
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* ── Create Job Dialog ── */}
      <Dialog
        open={createOpen}
        onOpenChange={(open) => {
          if (!open) setCreateOpen(false);
        }}
      >
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle>Submit New Spark Job</DialogTitle>
            <DialogDescription>
              Configure your batch Spark job parameters
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            {/* Name */}
            <div className="space-y-1.5">
              <Label htmlFor="job-name">
                Name <span className="text-destructive">*</span>
              </Label>
              <Input
                id="job-name"
                value={form.name}
                onChange={(e) =>
                  setForm((f) => ({ ...f, name: e.target.value }))
                }
                placeholder="My Spark Job"
              />
              {formErrors.name && (
                <p className="text-xs text-destructive">{formErrors.name}</p>
              )}
            </div>

            {/* Type */}
            <div className="space-y-1.5">
              <Label htmlFor="job-type">
                Type <span className="text-destructive">*</span>
              </Label>
              <Select
                value={form.type}
                onValueChange={(v) =>
                  setForm((f) => ({ ...f, type: v as SparkJobType }))
                }
              >
                <SelectTrigger id="job-type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="scala">Scala</SelectItem>
                  <SelectItem value="python">Python</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Main class (Scala only) */}
            {form.type === 'scala' && (
              <div className="space-y-1.5">
                <Label htmlFor="job-main-class">
                  Main Class <span className="text-destructive">*</span>
                </Label>
                <Input
                  id="job-main-class"
                  value={form.mainClass}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, mainClass: e.target.value }))
                  }
                  placeholder="com.example.MyApp"
                />
                {formErrors.mainClass && (
                  <p className="text-xs text-destructive">
                    {formErrors.mainClass}
                  </p>
                )}
              </div>
            )}

            {/* Main app file */}
            <div className="space-y-1.5">
              <Label htmlFor="job-app-file">
                Main App File <span className="text-destructive">*</span>
              </Label>
              <Input
                id="job-app-file"
                value={form.mainAppFile}
                onChange={(e) =>
                  setForm((f) => ({ ...f, mainAppFile: e.target.value }))
                }
                placeholder={
                  form.type === 'scala'
                    ? 's3a://bucket/path/to/app.jar'
                    : 's3a://bucket/path/to/app.py'
                }
              />
              {formErrors.mainAppFile && (
                <p className="text-xs text-destructive">
                  {formErrors.mainAppFile}
                </p>
              )}
            </div>

            {/* Arguments */}
            <div className="space-y-1.5">
              <Label htmlFor="job-args">Arguments</Label>
              <Textarea
                id="job-args"
                value={form.arguments}
                onChange={(e) =>
                  setForm((f) => ({ ...f, arguments: e.target.value }))
                }
                placeholder="--input /data/input --output /data/output"
                rows={2}
              />
            </div>

            {/* Resources */}
            <div>
              <Label className="mb-2 block">Resources</Label>
              <div className="grid grid-cols-3 gap-3">
                <div className="space-y-1.5">
                  <Label htmlFor="job-cpu" className="text-xs">
                    CPU
                  </Label>
                  <Input
                    id="job-cpu"
                    value={form.cpu}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, cpu: e.target.value }))
                    }
                    placeholder="1"
                  />
                  {formErrors.cpu && (
                    <p className="text-xs text-destructive">
                      {formErrors.cpu}
                    </p>
                  )}
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="job-memory" className="text-xs">
                    Memory
                  </Label>
                  <Input
                    id="job-memory"
                    value={form.memory}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, memory: e.target.value }))
                    }
                    placeholder="1g"
                  />
                  {formErrors.memory && (
                    <p className="text-xs text-destructive">
                      {formErrors.memory}
                    </p>
                  )}
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="job-executors" className="text-xs">
                    Executors
                  </Label>
                  <Input
                    id="job-executors"
                    type="number"
                    min={1}
                    value={form.executors}
                    onChange={(e) =>
                      setForm((f) => ({
                        ...f,
                        executors: Math.max(1, Number(e.target.value) || 1),
                      }))
                    }
                  />
                  {formErrors.executors && (
                    <p className="text-xs text-destructive">
                      {formErrors.executors}
                    </p>
                  )}
                </div>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setCreateOpen(false)}
              disabled={creating}
            >
              Cancel
            </Button>
            <Button onClick={handleCreate} disabled={creating}>
              {creating && (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              )}
              <Play className="h-4 w-4 mr-2" />
              Submit
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ── Logs Dialog ── */}
      <Dialog
        open={!!logsJob}
        onOpenChange={(open) => {
          if (!open) setLogsJob(null);
        }}
      >
        <DialogContent className="sm:max-w-[720px] max-h-[80vh]">
          <DialogHeader>
            <DialogTitle>
              Logs — {logsJob?.name ?? ''}
            </DialogTitle>
            <DialogDescription>
              {logsJob && (
                <span className="flex items-center gap-2">
                  {jobStatusBadge(logsJob.status)}
                  <span className="text-xs text-muted-foreground">
                    ID: {logsJob.id}
                  </span>
                </span>
              )}
            </DialogDescription>
          </DialogHeader>
          <ScrollArea className="h-[400px] rounded-md border bg-black/80 p-4 font-mono text-sm text-green-400">
            {logsLoading ? (
              <div className="flex items-center justify-center h-full">
                <Loader2 className="h-6 w-6 animate-spin" />
              </div>
            ) : (
              <pre className="whitespace-pre-wrap break-all">
                {logsContent}
              </pre>
            )}
          </ScrollArea>
          <DialogFooter>
            <Button variant="outline" onClick={() => setLogsJob(null)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ── Stop Confirmation ── */}
      <AlertDialog
        open={!!stopTarget}
        onOpenChange={(open) => {
          if (!open) setStopTarget(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Stop "{stopTarget?.name ?? ''}"?
            </AlertDialogTitle>
            <AlertDialogDescription>
              This will cancel the running Spark job. Any in-progress work
              will be terminated and cannot be resumed.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={stopping}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={confirmStop}
              disabled={stopping}
            >
              {stopping && (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              )}
              Stop Job
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
