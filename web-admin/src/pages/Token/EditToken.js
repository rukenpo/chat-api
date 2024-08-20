import React, {useEffect, useState, useContext} from 'react';
import {API, isMobile, showError, showSuccess, timestamp2string, isAdmin} from '../../helpers';
import {renderQuota, renderQuotaWithPrompt} from '../../helpers/render';
import {Layout, SideSheet, Button, Space, Spin, Banner, Input, DatePicker, AutoComplete, Typography, Select} from "@douyinfe/semi-ui";
import Title from "@douyinfe/semi-ui/lib/es/typography/title";
import {Divider} from "semantic-ui-react";
import {UserContext} from '../../context/User';

const EditToken = (props) => {
    const isEdit = props.editingToken.id !== undefined;
    const [loading, setLoading] = useState(isEdit);
    const isAdminUser = isAdmin();
    const [groupOptions, setGroupOptions] = useState([]);
    const [userState] = useContext(UserContext);
    const [selectedGroup, setSelectedGroup] = useState();
    const [billingStrategy, setBillingStrategy] = useState('0');
    const [modelRatioEnabled, setModelRatioEnabled] = useState('');
    const [billingByRequestEnabled, setBillingByRequestEnabled] = useState('');
    const [options, setOptions] = useState({});
    const [models, setModels] = useState([]);
    const originInputs = {
        name: '',
        fixed_content: '',
        subnet: '',
        remain_quota: isEdit ? 0 : 500000,
        expired_time: -1,
        unlimited_quota: false,
        expiry_mode: 'fixed',
        duration: 24,
    };
    const [inputs, setInputs] = useState(originInputs);
    const {name, remain_quota, expired_time, unlimited_quota, fixed_content, subnet, expiry_mode, duration} = inputs;
    const [tokenCount, setTokenCount] = useState(1);

    const handleInputChange = (name, value) => {
        setInputs((inputs) => ({...inputs, [name]: value}));
    };

    const handleCancel = () => {
        props.handleClose();
    }

    const setExpiredTime = (month, day, hour, minute) => {
        let now = new Date();
        let timestamp = now.getTime() / 1000;
        let seconds = month * 30 * 24 * 60 * 60;
        seconds += day * 24 * 60 * 60;
        seconds += hour * 60 * 60;
        seconds += minute * 60;
        if (seconds !== 0) {
            timestamp += seconds;
            setInputs({...inputs, expired_time: timestamp2string(timestamp)});
        } else {
            setInputs({...inputs, expired_time: -1});
        }
    };

    const setUnlimitedQuota = () => {
        setInputs({...inputs, unlimited_quota: !unlimited_quota});
    };

    const loadToken = async () => {
        setLoading(true);
        let res = await API.get(`/api/token/${props.editingToken.id}`);
        const {success, message, data} = res.data;
        if (success) {
            if (data.expired_time !== -1) {
                data.expired_time = timestamp2string(data.expired_time);
            }
            if (data.models && typeof data.models === 'string') {
                data.models = data.models.split(',');
            }
            data.expiry_mode = data.expired_time === -1 ? 'first_use' : 'fixed';
            data.duration = data.duration ? data.duration / 3600 : 24;
            setInputs(data);
            setSelectedGroup(data.group || userState?.user?.group || 'default');
        } else {
            showError(message);
        }
        setLoading(false);
    };

    useEffect(() => {
        if (isEdit) {
            loadToken().then();
        } else {
            setInputs(originInputs);
            setSelectedGroup(userState?.user?.group || 'default');
        }
        if (isAdminUser) {
            fetchGroups().then();
        }
        getOptions();
        loadModels();
    }, [props.editingToken.id]);

    const getOptions = async () => {
        const res = await API.get('/api/user/option');
        const {success, message, data} = res.data;
        if (success) {
            let newOptions = {};
            data.forEach((item) => {
                newOptions[item.key] = item.value;
            });
            setOptions(newOptions);
        } else {
            showError(message);
        }
    };

    useEffect(() => {
        if (options.ModelRatioEnabled) {
            setModelRatioEnabled(options.ModelRatioEnabled === 'true');
        }
        if (options.BillingByRequestEnabled) {
            setBillingByRequestEnabled(options.BillingByRequestEnabled === 'true');
        }
    }, [options]);

    const handleTokenCountChange = (value) => {
        const count = parseInt(value, 10);
        if (!isNaN(count) && count > 0) {
            setTokenCount(count);
        }
    };

    const generateRandomSuffix = () => {
        const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
        let result = '';
        for (let i = 0; i < 6; i++) {
            result += characters.charAt(Math.floor(Math.random() * characters.length));
        }
        return result;
    };

    const submit = async () => {
        setLoading(true);
        try {
            if (isEdit) {
                let localInputs = {...inputs};
                if (localInputs.models && localInputs.models.length > 0) {
                    localInputs.models = localInputs.models.join(',');
                } else {
                    localInputs.models = '';
                }
                localInputs.remain_quota = parseInt(localInputs.remain_quota);
                localInputs.group = selectedGroup;
    
                if (localInputs.expiry_mode === 'fixed') {
                    if (localInputs.expired_time !== -1) {
                        let time = Date.parse(localInputs.expired_time);
                        if (isNaN(time)) {
                            throw new Error('过期时间格式错误！');
                        }
                        localInputs.expired_time = Math.ceil(time / 1000);
                    }
                } else {
                    localInputs.expired_time = -1;
                    localInputs.duration = parseInt(localInputs.duration)*3600;
                }
    
                let res = await API.put(`/api/token/`, {...localInputs, id: parseInt(props.editingToken.id)});
                const {success, message} = res.data;
                if (success) {
                    showSuccess('令牌更新成功！');
                    props.refresh();
                    props.handleClose();
                } else {
                    throw new Error(message);
                }
            } else {
                let successCount = 0;
                for (let i = 0; i < tokenCount; i++) {
                    let localInputs = {...inputs};
                    if (localInputs.models && localInputs.models.length > 0) {
                        localInputs.models = localInputs.models.join(',');
                    } else {
                        localInputs.models = '';
                    }
                    if (i !== 0) {
                        localInputs.name = `${inputs.name}-${generateRandomSuffix()}`;
                    }
                    localInputs.remain_quota = parseInt(localInputs.remain_quota);
                    localInputs.group = selectedGroup;
                    localInputs.billing_enabled = billingStrategy === '1';
                    if (localInputs.expiry_mode === 'fixed') {
                        if (localInputs.expired_time !== -1) {
                            let time = Date.parse(localInputs.expired_time);
                            if (isNaN(time)) {
                                throw new Error('过期时间格式错误！');
                            }
                            localInputs.expired_time = Math.ceil(time / 1000);
                        }
                    } else {
                        localInputs.expired_time = -1;
                        localInputs.duration = parseInt(localInputs.duration) * 3600;
                    }
    
                    let res = await API.post(`/api/token/`, localInputs);
                    const {success, message} = res.data;
    
                    if (success) {
                        successCount++;
                    } else {
                        throw new Error(message);
                    }
                }
    
                if (successCount > 0) {
                    showSuccess(`${successCount}个令牌创建成功，请在列表页面点击复制获取令牌！`);
                    props.refresh();
                    props.handleClose();
                }
            }
        } catch (error) {
            showError(error.message);
        } finally {
            setLoading(false);
            setInputs(originInputs);
            setTokenCount(1);
        }
    };
    

    const fetchGroups = async () => {
        try {
            let res = await API.get(`/api/group/`);
            setGroupOptions(res.data.data.map((group) => ({
                label: group,
                value: group,
            })));
        } catch (error) {
            showError(error.message);
        }
    };

    const loadModels = async () => {
        try {
            let res = await API.get('/api/user/models');
            const {success, message, data} = res.data;
            if (success) {
                setModels(data);
            } else {
                showError(message);
            }
        } catch (err) {
            showError(err.message);
        }
    };

    return (
        <>
            <SideSheet
                placement={isEdit ? 'right' : 'left'}
                title={<Title level={3}>{isEdit ? '更新令牌信息' : '创建新的令牌'}</Title>}
                headerStyle={{borderBottom: '1px solid var(--semi-color-border)'}}
                bodyStyle={{borderBottom: '1px solid var(--semi-color-border)'}}
                visible={props.visiable}
                footer={
                    <div style={{display: 'flex', justifyContent: 'flex-end'}}>
                        <Space>
                            <Button theme='solid' size={'large'} onClick={submit}>提交</Button>
                            <Button theme='solid' size={'large'} type={'tertiary'} onClick={handleCancel}>取消</Button>
                        </Space>
                    </div>
                }
                closeIcon={null}
                onCancel={() => handleCancel()}
                width={isMobile() ? '100%' : 600}
            >
                <Spin spinning={loading}>
                    <Input
                        style={{marginTop: 20}}
                        label='名称'
                        name='name'
                        placeholder={'请输入名称'}
                        onChange={(value) => handleInputChange('name', value)}
                        value={name}
                        autoComplete='new-password'
                        required={!isEdit}
                    />
                    <Divider/>
                    <Select
                        style={{marginTop: 20}}
                        label='过期模式'
                        name='expiry_mode'
                        placeholder={'请选择过期模式'}
                        onChange={(value) => handleInputChange('expiry_mode', value)}
                        value={expiry_mode}
                        disabled={isEdit}
                    >
                        <Select.Option value="fixed">固定时间</Select.Option>
                        <Select.Option value="first_use">首次使用后计时</Select.Option>
                    </Select>
                    {expiry_mode === 'fixed' && (
                        <>
                            <DatePicker
                                style={{marginTop: 20}}
                                label='过期时间'
                                name='expired_time'
                                placeholder={'请选择过期时间'}
                                onChange={(value) => handleInputChange('expired_time', value)}
                                value={expired_time}
                                autoComplete='new-password'
                                type='dateTime'
                            />
                            <div style={{marginTop: 20}}>
                                <Space>
                                    <Button type={'tertiary'} onClick={() => {
                                        setExpiredTime(0, 0, 0, 0);
                                    }}>永不过期</Button>
                                    <Button type={'tertiary'} onClick={() => {
                                        setExpiredTime(0, 0, 1, 0);
                                    }}>一小时</Button>
                                    <Button type={'tertiary'} onClick={() => {
                                        setExpiredTime(1, 0, 0, 0);
                                    }}>一个月</Button>
                                    <Button type={'tertiary'} onClick={() => {
                                        setExpiredTime(0, 1, 0, 0);
                                    }}>一天</Button>
                                </Space>
                            </div>
                        </>
                    )}
                    {expiry_mode === 'first_use' && (
                        <Input
                            style={{marginTop: 20}}
                            label='有效期（小时）'
                            name='duration'
                            placeholder={'请输入有效期小时数'}
                            onChange={(value) => handleInputChange('duration', value)}
                            value={duration}
                            autoComplete='new-password'
                            type='number'
                        />
                    )}
                    <Divider/>
                    <Banner type={'warning'}
                            description={'注意，令牌的额度仅用于限制令牌本身的最大额度使用量，实际的使用受到账户的剩余额度限制。'}></Banner>
                    <div style={{marginTop: 20}}>
                        <Typography.Text>{`额度${renderQuotaWithPrompt(remain_quota)}`}</Typography.Text>
                    </div>
                    <AutoComplete
                        style={{marginTop: 8}}
                        name='remain_quota'
                        placeholder={'请输入额度'}
                        onChange={(value) => handleInputChange('remain_quota', value)}
                        value={remain_quota}
                        autoComplete='new-password'
                        type='number'
                        position={'top'}
                        data={[
                            {value: 500000, label: '1$'},
                            {value: 5000000, label: '10$'},
                            {value: 25000000, label: '50$'},
                            {value: 50000000, label: '100$'},
                            {value: 250000000, label: '500$'},
                            {value: 500000000, label: '1000$'},
                        ]}
                        disabled={unlimited_quota}
                    />
                    <div>
                        <Button style={{marginTop: 8}} type={'warning'} onClick={() => {
                            setUnlimitedQuota();
                        }}>{unlimited_quota ? '取消无限额度' : '设为无限额度'}</Button>
                    </div>
                    <Divider/>
                    <div style={{marginTop: 20}}>
                        <Typography.Text>可用模型</Typography.Text>
                    </div>
                    <Select
                        style={{marginTop: 20}}
                        label="可用模型"
                        placeholder={'请选择可用模型，为空代表全部可用'}
                        onChange={(value) => handleInputChange('models', value)}
                        value={inputs.models}
                        multiple
                        selection
                        autoComplete='new-password'
                        optionList={models.map(model => ({label: model, value: model}))}
                    >
                        {models.map(model => (
                            <Select.Option key={model} value={model}>
                                {model}
                            </Select.Option>
                        ))}
                    </Select>
                    <Divider/>

                    <div style={{display: 'flex', justifyContent: 'space-between', marginTop: 20}}>
                        {isAdminUser && (
                            <div style={{flex: 1, marginRight: 8}}>
                                <Typography.Text>选择分组</Typography.Text>
                                <Select
                                    placeholder={'请选择分组'}
                                    name='group'
                                    onChange={(value) => setSelectedGroup(value)}
                                    value={selectedGroup}
                                    autoComplete='new-password'
                                    optionList={groupOptions}
                                    style={{width: '100%', marginTop: 8}}
                                />
                            </div>
                        )}

                        {!isEdit && modelRatioEnabled && billingByRequestEnabled && (
                            <div style={{flex: 1, marginLeft: isAdminUser ? 8 : 0}}>
                                <Typography.Text>计费策略</Typography.Text>
                                <Select
                                    placeholder={'请选择计费策略'}
                                    value={billingStrategy}
                                    onChange={(value) => setBillingStrategy(value)}
                                    style={{width: '100%', marginTop: 8}}
                                >
                                    <Select.Option value="0">按Token计费</Select.Option>
                                    <Select.Option value="1">按次计费</Select.Option>
                                </Select>
                            </div>
                        )}
                    </div>
                    <Divider/>
                    <div style={{marginTop: 10}}>
                        <Typography.Text>IP 限制</Typography.Text>
                    </div>
                    <Input
                        style={{marginTop: 20}}
                        label='自定义'
                        name='subnet'
                        placeholder={'请输入允许访问的网段，例如：192.168.0.0/24'}
                        onChange={(value) => handleInputChange('subnet', value)}
                        value={subnet}
                        autoComplete='new-password'
                        required={!isEdit}
                    />
                    <Divider/>
                    <div style={{marginTop: 10}}>
                        <Typography.Text>自定义内容</Typography.Text>
                    </div>
                    <Input
                        style={{marginTop: 20}}
                        label='自定义'
                        name='fixed_content'
                        placeholder={'请输入自定义后缀'}
                        onChange={(value) => handleInputChange('fixed_content', value)}
                        value={fixed_content}
                        autoComplete='new-password'
                        required={!isEdit}
                    />
                    <Divider/>

                    {!isEdit && (
                        <>
                            <div style={{marginTop: 20}}>
                                <Typography.Text>新建数量</Typography.Text>
                            </div>
                            <AutoComplete
                                style={{marginTop: 8}}
                                label='数量'
                                placeholder={'请选择或输入创建令牌的数量'}
                                onChange={(value) => handleTokenCountChange(value)}
                                onSelect={(value) => handleTokenCountChange(value)}
                                value={tokenCount.toString()}
                                autoComplete='off'
                                type='number'
                                data={[
                                    {value: 10, label: '10个'},
                                    {value: 20, label: '20个'},
                                    {value: 30, label: '30个'},
                                    {value: 100, label: '100个'},
                                ]}
                                disabled={unlimited_quota}
                            />
                        </>
                    )}
                </Spin>
            </SideSheet>
        </>
    );
};

export default EditToken;