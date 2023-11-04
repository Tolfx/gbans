import NiceModal from '@ebay/nice-modal-react';
import { AssetViewer } from './AssetViewer';
import { BanASNModal } from './BanASNModal';
import { BanCIDRModal } from './BanCIDRModal';
import { BanGroupModal } from './BanGroupModal';
import { BanSteamModal } from './BanSteamModal';
import { ConfirmDeleteFilterModal } from './ConfirmDeleteFilterModal';
import { ConfirmationModal } from './ConfirmationModal';
import { ContestEditor } from './ContestEditor';
import { ContestEntryDeleteModal } from './ContestEntryDeleteModal';
import { ContestEntryModal } from './ContestEntryModal';
import { FileUploadModal } from './FileUploadModal';
import { FilterEditModal } from './FilterEditModal';
import { MessageContextModal } from './MessageContextModal';
import { ServerDeleteModal } from './ServerDeleteModal';
import { ServerEditorModal } from './ServerEditorModal';
import { UnbanASNModal } from './UnbanASNModal';
import { UnbanCIDRModal } from './UnbanCIDRModal';
import { UnbanGroupModal } from './UnbanGroupModal';
import { UnbanSteamModal } from './UnbanSteamModal';

export const ModalContestEditor = 'modal-contest-editor';
export const ModalContestEntry = 'modal-contest-entry';
export const ModalContestEntryDelete = 'modal-contest-entry-delete';
export const ModalConfirm = 'modal-confirm';
export const ModalAssetViewer = 'modal-asset-viewer';
export const ModalBanSteam = 'modal-ban-steam';
export const ModalBanASN = 'modal-ban-asn';
export const ModalBanCIDR = 'modal-ban-cidr';
export const ModalBanGroup = 'modal-ban-group';
export const ModalUnbanSteam = 'modal-unban-steam';
export const ModalUnbanASN = 'modal-unban-asn';
export const ModalUnbanCIDR = 'modal-unban-cidr';
export const ModalUnbanGroup = 'modal-unban-group';
export const ModalServerEditor = 'modal-server-editor';
export const ModalServerDelete = 'modal-server-delete';
export const ModalMessageContext = 'modal-message-context';
export const ModalFileUpload = 'modal-file-upload';
export const ModalFilterDelete = 'modal-filter-delete';
export const ModalFilterEditor = 'modal-filter-editor';

NiceModal.register(ModalContestEntryDelete, ContestEntryDeleteModal);
NiceModal.register(ModalContestEditor, ContestEditor);
NiceModal.register(ModalContestEntry, ContestEntryModal);
NiceModal.register(ModalAssetViewer, AssetViewer);
NiceModal.register(ModalConfirm, ConfirmationModal);
NiceModal.register(ModalServerEditor, ServerEditorModal);
NiceModal.register(ModalServerDelete, ServerDeleteModal);
NiceModal.register(ModalMessageContext, MessageContextModal);
NiceModal.register(ModalFileUpload, FileUploadModal);
NiceModal.register(ModalFilterDelete, ConfirmDeleteFilterModal);
NiceModal.register(ModalFilterEditor, FilterEditModal);
NiceModal.register(ModalBanSteam, BanSteamModal);
NiceModal.register(ModalBanASN, BanASNModal);
NiceModal.register(ModalBanCIDR, BanCIDRModal);
NiceModal.register(ModalBanGroup, BanGroupModal);
NiceModal.register(ModalUnbanSteam, UnbanSteamModal);
NiceModal.register(ModalUnbanASN, UnbanASNModal);
NiceModal.register(ModalUnbanCIDR, UnbanCIDRModal);
NiceModal.register(ModalUnbanGroup, UnbanGroupModal);
