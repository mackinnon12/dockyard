<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useConfirm } from 'primevue/useconfirm';
import { useToast } from 'primevue/usetoast';

import { CreateMySQL, DeleteContainer, ListContainers, RestartContainer, StartContainer, StopContainer } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { useLayout } from '../layout/composables/layout';

const confirm = useConfirm();
const toast = useToast();
const { toggleDarkMode, isDarkTheme } = useLayout();

const containers = ref([]);
const loading = ref(true);
const loadError = ref('');
const pending = ref({});
const showCreateDialog = ref(false);
const creating = ref(false);
const createError = ref('');
const creationProgress = ref(null);
const createForm = ref({
    name: 'mysql',
    version: '8.4',
    port: 3306,
    rootPassword: '',
    database: 'app'
});
let stopProgressListener;

const runningCount = computed(() => containers.value.filter((container) => container.state === 'running').length);
const stoppedCount = computed(() => containers.value.length - runningCount.value);

const shortId = (id) => id.slice(0, 12);
const isRunning = (container) => container.state === 'running';
const isPending = (id) => Boolean(pending.value[id]);
const progressPercent = computed(() => {
    if (!creationProgress.value) return 0;
    if (creationProgress.value.status === 'complete' || (creationProgress.value.stage === 'ready' && creationProgress.value.progress === '100%')) return 100;
    const match = creationProgress.value.progress?.match(/(\d+(?:\.\d+)?)%/);
    return match ? Math.min(100, Number(match[1])) : 0;
});
const progressLabel = computed(() => {
    const progress = creationProgress.value;
    if (!progress) return 'Preparing MySQL';
    if (progress.message) return progress.message;
    if (progress.status) return progress.status;
    return 'Working';
});

const errorMessage = (error) => {
    if (typeof error === 'string') return error;
    return error?.message || 'Docker could not complete the request.';
};

const refreshContainers = async ({ quiet = false } = {}) => {
    if (!quiet) loading.value = true;
    loadError.value = '';

    try {
        containers.value = (await ListContainers()) || [];
    } catch (error) {
        loadError.value = errorMessage(error);
    } finally {
        loading.value = false;
    }
};

const runAction = async (container, label, action) => {
    pending.value = { ...pending.value, [container.id]: label };

    try {
        await action(container.id);
        await refreshContainers({ quiet: true });
        toast.add({ severity: 'success', summary: `${label} Complete`, detail: container.name, life: 2500 });
    } catch (error) {
        toast.add({ severity: 'error', summary: `${label} Failed`, detail: errorMessage(error), life: 5000 });
    } finally {
        const nextPending = { ...pending.value };
        delete nextPending[container.id];
        pending.value = nextPending;
    }
};

const confirmDelete = (container) => {
    confirm.require({
        header: 'Delete Container',
        message: `Delete ${container.name}? This cannot be undone.`,
        icon: 'pi pi-exclamation-triangle',
        rejectLabel: 'Cancel',
        acceptLabel: 'Delete',
        acceptClass: 'p-button-danger',
        accept: () => runAction(container, 'Delete', DeleteContainer)
    });
};

const openCreateDialog = () => {
    createError.value = '';
    creationProgress.value = null;
    showCreateDialog.value = true;
};

const closeCreateDialog = () => {
    if (!creating.value) showCreateDialog.value = false;
};

const createMySQL = async () => {
    createError.value = '';
    creationProgress.value = { stage: 'starting', status: 'starting', message: 'Preparing MySQL' };
    creating.value = true;

    try {
        await CreateMySQL({ ...createForm.value });
        await refreshContainers({ quiet: true });
        toast.add({ severity: 'success', summary: 'MySQL Created', detail: `${createForm.value.name} is ready`, life: 3500 });
        showCreateDialog.value = false;
        createForm.value.rootPassword = '';
    } catch (error) {
        createError.value = errorMessage(error);
        toast.add({ severity: 'error', summary: 'MySQL Creation Failed', detail: createError.value, life: 6000 });
    } finally {
        creating.value = false;
    }
};

onMounted(() => {
    refreshContainers();
    stopProgressListener = EventsOn('mysql:creation-progress', (progress) => {
        creationProgress.value = Array.isArray(progress) ? progress[0] : progress;
    });
});

onBeforeUnmount(() => stopProgressListener?.());
</script>

<template>
    <main class="dockyard-shell" :class="{ 'is-dark': isDarkTheme }">
        <Toast />
        <ConfirmDialog />

        <section class="workspace" aria-labelledby="page-title">
            <header class="page-header">
                <div>
                    <h1 id="page-title">Docker Containers</h1>
                    <p>Manage the database containers running on this machine.</p>
                </div>
                <div class="header-actions">
                    <Button label="Create MySQL" icon="pi pi-plus" @click="openCreateDialog" />
                    <Button label="Refresh" icon="pi pi-refresh" severity="secondary" outlined :loading="loading" @click="refreshContainers()" />
                    <Button
                        :label="isDarkTheme ? 'Light Mode' : 'Dark Mode'"
                        :icon="isDarkTheme ? 'pi pi-sun' : 'pi pi-moon'"
                        severity="secondary"
                        outlined
                        :aria-label="isDarkTheme ? 'Switch to light mode' : 'Switch to dark mode'"
                        @click="toggleDarkMode"
                    />
                </div>
            </header>

            <div class="summary-grid" aria-label="Container summary">
                <div class="summary-card">
                    <span>Total</span><strong>{{ containers.length }}</strong>
                </div>
                <div class="summary-card">
                    <span>Running</span><strong class="running-count">{{ runningCount }}</strong>
                </div>
                <div class="summary-card">
                    <span>Stopped</span><strong>{{ stoppedCount }}</strong>
                </div>
            </div>

            <section class="container-panel" aria-label="Containers">
                <div v-if="loading" class="state-panel">
                    <ProgressSpinner strokeWidth="4" />
                    <p>Loading Docker containers…</p>
                </div>

                <div v-else-if="loadError" class="state-panel error-state">
                    <i class="pi pi-exclamation-circle"></i>
                    <h2>Docker Is Unavailable</h2>
                    <p>{{ loadError }}</p>
                    <Button label="Try Again" icon="pi pi-refresh" @click="refreshContainers()" />
                </div>

                <div v-else-if="containers.length === 0" class="state-panel">
                    <i class="pi pi-inbox"></i>
                    <h2>No Containers Found</h2>
                    <p>Docker is connected, but there are no containers to manage yet.</p>
                </div>

                <DataTable v-else :value="containers" dataKey="id" stripedRows responsiveLayout="scroll">
                    <Column field="name" header="Name">
                        <template #body="{ data }">
                            <div class="container-name">
                                <span class="container-icon"><i class="pi pi-database"></i></span>
                                <div>
                                    <strong>{{ data.name }}</strong>
                                    <small>{{ shortId(data.id) }}</small>
                                </div>
                            </div>
                        </template>
                    </Column>
                    <Column field="image" header="Image"></Column>
                    <Column header="Public Address">
                        <template #body="{ data }">
                            <code v-if="data.publicAddress">{{ data.publicAddress }}</code>
                            <span v-else class="muted-value">—</span>
                        </template>
                    </Column>
                    <Column header="Port">
                        <template #body="{ data }">
                            <code v-if="data.publicPort">{{ data.publicPort }}</code>
                            <span v-else class="muted-value">—</span>
                        </template>
                    </Column>
                    <Column header="Status">
                        <template #body="{ data }">
                            <div class="status-cell">
                                <Tag :value="isRunning(data) ? 'Running' : 'Stopped'" :severity="isRunning(data) ? 'success' : 'secondary'" />
                                <small>{{ data.status }}</small>
                            </div>
                        </template>
                    </Column>
                    <Column header="Actions" class="actions-column">
                        <template #body="{ data }">
                            <div class="action-group">
                                <Button
                                    v-if="!isRunning(data)"
                                    icon="pi pi-play"
                                    severity="success"
                                    text
                                    rounded
                                    :loading="pending[data.id] === 'Start'"
                                    :disabled="isPending(data.id)"
                                    :aria-label="`Start ${data.name}`"
                                    @click="runAction(data, 'Start', StartContainer)"
                                />
                                <Button v-else icon="pi pi-stop" severity="warn" text rounded :loading="pending[data.id] === 'Stop'" :disabled="isPending(data.id)" :aria-label="`Stop ${data.name}`" @click="runAction(data, 'Stop', StopContainer)" />
                                <Button
                                    icon="pi pi-refresh"
                                    severity="secondary"
                                    text
                                    rounded
                                    :loading="pending[data.id] === 'Restart'"
                                    :disabled="isPending(data.id)"
                                    :aria-label="`Restart ${data.name}`"
                                    @click="runAction(data, 'Restart', RestartContainer)"
                                />
                                <Button icon="pi pi-trash" severity="danger" text rounded :loading="pending[data.id] === 'Delete'" :disabled="isPending(data.id)" :aria-label="`Delete ${data.name}`" @click="confirmDelete(data)" />
                            </div>
                        </template>
                    </Column>
                </DataTable>
            </section>
        </section>

        <Dialog v-model:visible="showCreateDialog" modal header="Create MySQL Database" :style="{ width: 'min(34rem, calc(100vw - 2rem))' }" :closable="!creating" :dismissableMask="!creating">
            <form class="create-form" @submit.prevent="createMySQL">
                <p class="dialog-intro">Pull a MySQL image and create a persistent Docker database container.</p>

                <div class="form-grid">
                    <div class="field">
                        <label for="mysql-name">Name</label>
                        <InputText id="mysql-name" v-model="createForm.name" :disabled="creating" required autocomplete="off" />
                        <small>Letters, numbers, hyphens, and underscores.</small>
                    </div>
                    <div class="field">
                        <label for="mysql-version">Version</label>
                        <Select id="mysql-version" v-model="createForm.version" :options="['8.4']" :disabled="creating" />
                    </div>
                    <div class="field">
                        <label for="mysql-port">Host Port</label>
                        <InputNumber id="mysql-port" v-model="createForm.port" :min="1" :max="65535" :disabled="creating" :useGrouping="false" fluid />
                    </div>
                    <div class="field">
                        <label for="mysql-database">Database</label>
                        <InputText id="mysql-database" v-model="createForm.database" :disabled="creating" required autocomplete="off" />
                    </div>
                    <div class="field field-full">
                        <label for="mysql-password">Root Password</label>
                        <Password id="mysql-password" v-model="createForm.rootPassword" :disabled="creating" :feedback="false" toggleMask fluid required autocomplete="new-password" />
                    </div>
                </div>

                <Message v-if="createError" severity="error" :closable="false">{{ createError }}</Message>

                <div v-if="creating || creationProgress" class="creation-progress">
                    <div class="progress-heading">
                        <span>{{ progressLabel }}</span>
                        <span>{{ progressPercent }}%</span>
                    </div>
                    <ProgressBar :value="progressPercent" :showValue="false" />
                    <div v-if="creationProgress" class="progress-details">
                        <span v-if="creationProgress.status">Status: {{ creationProgress.status }}</span>
                        <span v-if="creationProgress.id">ID: {{ creationProgress.id }}</span>
                        <span v-if="creationProgress.progress">Progress: {{ creationProgress.progress }}</span>
                    </div>
                </div>

                <div class="dialog-actions">
                    <Button type="button" label="Cancel" severity="secondary" text :disabled="creating" @click="closeCreateDialog" />
                    <Button type="submit" label="Create Database" icon="pi pi-database" :loading="creating" :disabled="!createForm.rootPassword" />
                </div>
            </form>
        </Dialog>
    </main>
</template>

<style scoped>
.dockyard-shell {
    min-height: 100vh;
    background: #f6f7f9;
    color: #18212f;
    padding: clamp(1.5rem, 4vw, 4rem);
}

.workspace {
    width: min(1180px, 100%);
    margin: 0 auto;
}

.page-header {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 2rem;
    margin-bottom: 2rem;
}

.header-actions {
    display: flex;
    gap: 0.75rem;
}

.eyebrow {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    color: #64748b;
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
}

.page-header h1 {
    color: #111827;
    font-size: clamp(2rem, 4vw, 3.25rem);
    letter-spacing: -0.04em;
    line-height: 1;
    margin: 0.75rem 0;
}

.page-header p {
    color: #64748b;
    font-size: 1.05rem;
}

.summary-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 1rem;
    margin-bottom: 1rem;
}

.summary-card,
.container-panel {
    background: #ffffff;
    border: 1px solid #e5e7eb;
    border-radius: 1rem;
    box-shadow: 0 8px 30px rgba(15, 23, 42, 0.04);
}

.summary-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1.25rem 1.5rem;
}

.summary-card span {
    color: #64748b;
    font-weight: 600;
}
.summary-card strong {
    font-size: 1.75rem;
}
.summary-card .running-count {
    color: #16a34a;
}
.container-panel {
    overflow: hidden;
}

.state-panel {
    display: flex;
    min-height: 360px;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 3rem;
    text-align: center;
}

.state-panel > .pi {
    color: #94a3b8;
    font-size: 2.5rem;
    margin-bottom: 1rem;
}
.state-panel h2 {
    color: #1f2937;
    margin: 0 0 0.5rem;
}
.state-panel p {
    color: #64748b;
    max-width: 520px;
    margin-bottom: 1.5rem;
}
.error-state > .pi {
    color: #dc2626;
}

.container-name,
.status-cell,
.action-group {
    display: flex;
    align-items: center;
}
.container-name {
    gap: 0.85rem;
}
.container-name > div:last-child,
.status-cell {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.3rem;
}
.container-name small,
.status-cell small {
    color: #94a3b8;
}

.muted-value {
    color: #94a3b8;
}

code {
    color: #475569;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 0.85rem;
}

.container-icon {
    display: grid;
    width: 2.4rem;
    height: 2.4rem;
    place-items: center;
    border-radius: 0.7rem;
    background: #eef2ff;
    color: #4f46e5;
}

.action-group {
    justify-content: flex-end;
    gap: 0.15rem;
}

.create-form {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
}

.dialog-intro {
    color: #64748b;
    margin: 0;
}

.form-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
}

.field {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
}

.field-full {
    grid-column: 1 / -1;
}

.field label {
    color: #334155;
    font-size: 0.875rem;
    font-weight: 600;
}

.field small {
    color: #94a3b8;
    font-size: 0.75rem;
}

.creation-progress {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    padding: 1rem;
    border: 1px solid #e2e8f0;
    border-radius: 0.75rem;
    background: #f8fafc;
}

.progress-heading,
.progress-details {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
}

.progress-heading {
    color: #334155;
    font-size: 0.875rem;
    font-weight: 600;
}

.progress-details {
    flex-wrap: wrap;
    color: #64748b;
    font-size: 0.75rem;
}

.dialog-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
}

.dockyard-shell.is-dark {
    background: #111827;
    color: #e5e7eb;
}

.dockyard-shell.is-dark .page-header h1,
.dockyard-shell.is-dark .state-panel h2 {
    color: #f8fafc;
}

.dockyard-shell.is-dark .page-header p,
.dockyard-shell.is-dark .summary-card span,
.dockyard-shell.is-dark .state-panel p,
.dockyard-shell.is-dark .dialog-intro,
.dockyard-shell.is-dark .field small,
.dockyard-shell.is-dark .container-name small,
.dockyard-shell.is-dark .status-cell small {
    color: #94a3b8;
}

.dockyard-shell.is-dark .summary-card,
.dockyard-shell.is-dark .container-panel {
    background: #1f2937;
    border-color: #374151;
    box-shadow: 0 8px 30px rgba(0, 0, 0, 0.2);
}

.dockyard-shell.is-dark .field label,
.dockyard-shell.is-dark .progress-heading {
    color: #e5e7eb;
}

.dockyard-shell.is-dark .creation-progress {
    background: #111827;
    border-color: #374151;
}

.dockyard-shell.is-dark .progress-details,
.dockyard-shell.is-dark .muted-value,
.dockyard-shell.is-dark code {
    color: #cbd5e1;
}

.dockyard-shell.is-dark .container-icon {
    background: #312e81;
    color: #c7d2fe;
}

:deep(.actions-column) {
    text-align: right;
}

@media (max-width: 680px) {
    .dockyard-shell {
        padding: 1.25rem;
    }
    .page-header {
        align-items: flex-start;
        flex-direction: column;
    }
    .summary-grid {
        grid-template-columns: 1fr;
    }
    .header-actions,
    .header-actions :deep(.p-button) {
        width: 100%;
    }
    .header-actions {
        flex-direction: column;
    }
    .form-grid {
        grid-template-columns: 1fr;
    }
    .field-full {
        grid-column: auto;
    }
}
</style>
